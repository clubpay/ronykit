package ee

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/store"
)

const (
	outboxBatch = 100
	outboxLease = 30 * time.Second
)

// errPermanent marks failures that redelivery cannot fix (e.g. an undecodable
// payload); dispatch logs and acknowledges them instead of retrying forever.
type errPermanent struct{ err error }

func (e errPermanent) Error() string { return e.err.Error() }
func (e errPermanent) Unwrap() error { return e.err }

// view runs fn in a read-only unit of work, which the store may serve from a
// replica.
func (e *Env) view(ctx context.Context, allow map[reflect.Type]bool, fn func(sc *scope) error) error {
	return e.store.View(ctx, func(ctx context.Context, tx store.Tx) error {
		return fn(newScope(ctx, e, tx, allow))
	})
}

// unit runs fn as one unit of work. With retry, the whole unit is re-run on
// ErrConflict, up to the Env's max retries.
func (e *Env) unit(
	ctx context.Context,
	allow map[reflect.Type]bool,
	retry bool,
	fn func(sc *scope) error,
) error {
	attempts := 1
	if retry && e.maxRetries > 0 {
		attempts = min(e.maxRetries, math.MaxInt-1) + 1
	}

	var err error

	for range attempts {
		err = e.store.Tx(ctx, func(ctx context.Context, tx store.Tx) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("ee: panic: %v", r)
				}
			}()

			sc := newScope(ctx, e, tx, allow)
			if err := fn(sc); err != nil {
				return err
			}

			return sc.commit()
		})
		if !errors.Is(err, ErrConflict) {
			break
		}
	}

	if err == nil {
		e.kickRelay()
	}

	return err
}

// dispatch is the bus handler shared by state and aggregate controllers.
func (e *Env) dispatch(
	ctx context.Context,
	owner string,
	actions map[string]*action,
	allow map[reflect.Type]bool,
	msg bus.Message,
) error {
	a, ok := actions[msg.Event]
	if !ok {
		return nil
	}

	log := e.log.With("event", msg.Event, "id", msg.ID, "owner", owner)

	if msg.Expired(e.now()) {
		log.DebugContext(ctx, "ee: dropping expired event")

		return nil
	}

	if a.filter != nil && !a.filter.Match(msg.Labels) {
		return nil
	}

	err := e.unit(ctx, allow, true, func(sc *scope) error {
		h := e.chain(func(ctx context.Context, _ *Delivery) error {
			sc.ctx = ctx // middlewares may have enriched it (tracing, ...)

			return a.run(sc, &msg)
		})

		return h(sc.ctx, &Delivery{Meta: metaOf(&msg.Envelope), Owner: owner, Tx: sc.tx})
	})

	if _, ok := errors.AsType[errPermanent](err); ok {
		log.ErrorContext(ctx, "ee: dropping event", "error", err)

		return nil
	}

	return err
}

func (e *Env) kickRelay() {
	select {
	case e.kick <- struct{}{}:
	default:
	}
}

// relay moves committed outbox entries to the bus. Entries are acknowledged
// only after a successful publish; on failure their lease expires and they
// are retried (at-least-once).
func (e *Env) relay(ctx context.Context) {
	t := time.NewTicker(e.outboxPoll)
	defer t.Stop()

	for {
		e.drainOutbox(ctx)

		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case <-e.kick:
		}
	}
}

func (e *Env) drainOutbox(ctx context.Context) {
	ob := e.store.Outbox()

	for ctx.Err() == nil {
		entries, err := ob.Claim(ctx, outboxBatch, outboxLease)
		if err != nil {
			e.log.ErrorContext(ctx, "ee: outbox claim", "error", err)

			return
		}

		if len(entries) == 0 {
			return
		}

		ids := make([]string, 0, len(entries))
		envs := make([]bus.Envelope, 0, len(entries))

		for _, entry := range entries {
			ids = append(ids, entry.ID)

			var env bus.Envelope
			if err := json.Unmarshal(entry.Data, &env); err != nil {
				e.log.ErrorContext(ctx, "ee: dropping undecodable outbox entry", "id", entry.ID, "error", err)

				continue
			}

			envs = append(envs, env)
		}

		if err := e.bus.Publish(ctx, envs...); err != nil {
			e.log.ErrorContext(ctx, "ee: outbox publish", "error", err)

			return
		}

		if err := ob.Ack(ctx, ids...); err != nil {
			e.log.ErrorContext(ctx, "ee: outbox ack", "error", err)

			return
		}

		if len(entries) < outboxBatch {
			return
		}
	}
}
