package ee

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"runtime/debug"
	"time"

	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/store"
)

const (
	defaultOutboxBatch = 100
	defaultOutboxLease = 30 * time.Second
)

// errPermanent marks failures that redelivery cannot fix (e.g. an undecodable
// payload); dispatch logs and acknowledges them instead of retrying forever.
type errPermanent struct{ err error }

func (e errPermanent) Error() string { return e.err.Error() }
func (e errPermanent) Unwrap() error { return e.err }

// errPanic is a handler panic. The unit of work is rolled back and the error
// is returned so the bus redelivers up to its own attempt limit.
type errPanic struct{ err error }

func (e errPanic) Error() string { return e.err.Error() }
func (e errPanic) Unwrap() error { return e.err }

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

	for attempt := 1; attempt <= attempts; attempt++ {
		err = e.store.Tx(ctx, func(ctx context.Context, tx store.Tx) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = errPanic{fmt.Errorf("ee: panic: %v", r)}
					e.log.ErrorContext(ctx, "ee: panic in unit of work",
						"panic", r, "stack", string(debug.Stack()))
					e.observe(ctx, Observation{Kind: ObsPanic, Err: err})
				}
			}()

			sc := newScope(ctx, e, tx, allow)
			if err := fn(sc); err != nil {
				return err
			}

			return sc.commit()
		})
		if err == nil || !errors.Is(err, ErrConflict) {
			break
		}

		e.log.DebugContext(ctx, "ee: version conflict", "attempt", attempt, "of", attempts)
		e.observe(ctx, Observation{Kind: ObsConflict, Attempt: attempt, Err: err})

		if attempt == attempts && attempts > 1 {
			err = fmt.Errorf("ee: %w after %d attempts", err, attempts)
		}
	}

	if err == nil {
		e.kickRelay()
	}

	return err
}

// dispatch is the bus handler shared by entity and aggregate controllers.
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
		e.observe(ctx, Observation{
			Kind: ObsDropped, Err: err, Event: msg.Event, ID: msg.ID, Owner: owner,
		})

		return nil
	}

	if err != nil {
		if _, ok := errors.AsType[errPanic](err); !ok {
			e.observe(ctx, Observation{
				Kind: ObsDeliveryError, Err: err, Event: msg.Event, ID: msg.ID, Owner: owner,
			})
		}
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
		entries, err := ob.Claim(ctx, e.outboxBatch, e.outboxLease)
		if err != nil {
			e.log.ErrorContext(ctx, "ee: outbox claim", "error", err)
			e.observe(ctx, Observation{Kind: ObsOutboxError, Err: err})

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
				e.observe(ctx, Observation{Kind: ObsOutboxDropped, Err: err, ID: entry.ID})

				continue
			}

			envs = append(envs, env)
		}

		if len(envs) > 0 {
			if err := e.bus.Publish(ctx, envs...); err != nil {
				e.log.ErrorContext(ctx, "ee: outbox publish", "error", err)
				e.observe(ctx, Observation{Kind: ObsOutboxError, Err: err, Count: len(envs)})

				return
			}

			e.observe(ctx, Observation{Kind: ObsOutboxPublished, Count: len(envs)})
		}

		if err := ob.Ack(ctx, ids...); err != nil {
			e.log.ErrorContext(ctx, "ee: outbox ack", "error", err)
			e.observe(ctx, Observation{Kind: ObsOutboxError, Err: err, Count: len(ids)})

			return
		}

		if len(entries) < e.outboxBatch {
			return
		}
	}
}
