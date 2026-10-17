package ee

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/clubpay/ronykit/ee/store"
)

// Delivery describes the event an action is about to handle.
type Delivery struct {
	Meta

	// Owner is the consuming controller, e.g. "state:order".
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
// Markers are never removed by ee; expire them with backend tooling.
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

		// MarshalJSON fails only for years outside [0, 9999], which time.Now
		// cannot produce.
		at, _ := time.Now().UTC().MarshalJSON()

		return d.Tx.Insert(ctx, store.Record{Kind: DedupKind, ID: key, Version: 1, Data: at})
	}
}
