package ee

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/clubpay/ronykit/ee/store"
)

// Delivery describes the event an action is about to handle.
type Delivery struct {
	Meta

	// Owner is the consuming controller, e.g. "entity:order".
	Owner string
	// Tx is the transaction of the unit of work. Writes made through it
	// commit or roll back together with the handler's. Middlewares are free
	// to ignore it and use their own resources instead, at the cost of that
	// atomicity.
	Tx store.Tx
}

// Next continues the middleware chain; the last Next runs the action.
type Next func(ctx context.Context, d *Delivery) error

// Middleware wraps action handling. It runs inside the unit of work, after
// expiry and Filter checks. Returning nil without calling next skips the
// handler and acknowledges the event; returning an error rolls the unit of
// work back and asks the bus to redeliver.
type Middleware func(next Next) Next

// WithMiddleware appends middlewares; the first one is the outermost.
func WithMiddleware(mw ...Middleware) Option {
	return func(e *Env) { e.middlewares = append(e.middlewares, mw...) }
}

func (e *Env) chain(final Next) Next {
	h := final
	for _, v := range slices.Backward(e.middlewares) {
		h = v(h)
	}

	return h
}

// DedupKind is the store kind Dedup writes its markers under.
const DedupKind = "ee.dedup"

// Dedup skips events a controller already handled, by envelope ID. The
// marker is written in the same transaction as the handler's changes, so an
// event's effects are applied exactly once even though delivery is
// at-least-once. Two replicas racing on the same duplicate collide on the
// marker insert; the loser's unit of work is retried and then skipped.
//
// Markers are not removed as events are handled. Call Env.ExpireDedup on a
// schedule to drop markers older than your redelivery window.
func Dedup(next Next) Next {
	return func(ctx context.Context, d *Delivery) error {
		key := d.Owner + "/" + d.ID

		_, err := d.Tx.Get(ctx, DedupKind, key)
		switch {
		case err == nil:
			return nil
		case !errors.Is(err, store.ErrNotFound):
			return err
		}

		if err := next(ctx, d); err != nil {
			return err
		}

		payload, err := json.Marshal(struct {
			At time.Time `json:"at"`
		}{At: time.Now().UTC()})
		if err != nil {
			return err
		}

		return d.Tx.Insert(ctx, store.Record{Kind: DedupKind, ID: key, Version: 1, Data: payload})
	}
}

// ExpireDedup deletes Dedup markers whose timestamp is strictly before
// before, plus markers whose payload is not a timestamp. It returns how many
// records were deleted. Run it outside the request path; a day or more past
// your bus redelivery window is a safe cutoff.
func (e *Env) ExpireDedup(ctx context.Context, before time.Time) (int, error) {
	if e.store == nil {
		return 0, errors.New("ee: store is required")
	}

	var n int

	err := e.store.Tx(ctx, func(ctx context.Context, tx store.Tx) error {
		type doomed struct {
			id      string
			version uint64
		}

		var drop []doomed

		for rec, err := range tx.Find(ctx, store.Query{Kind: DedupKind}) {
			if err != nil {
				return err
			}

			at, ok := dedupTime(rec.Data)
			if !ok || at.Before(before) {
				drop = append(drop, doomed{rec.ID, rec.Version})
			}
		}

		for _, d := range drop {
			if err := tx.Delete(ctx, DedupKind, d.id, d.version); err != nil {
				return err
			}

			n++
		}

		return nil
	})

	return n, err
}

func dedupTime(data []byte) (time.Time, bool) {
	var marker struct {
		At time.Time `json:"at"`
	}
	if err := json.Unmarshal(data, &marker); err == nil && !marker.At.IsZero() {
		return marker.At, true
	}

	var at time.Time
	if err := json.Unmarshal(data, &at); err != nil || at.IsZero() {
		return time.Time{}, false
	}

	return at, true
}
