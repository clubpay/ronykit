package ee

import (
	"context"
	"fmt"

	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/store"
)

// action is the untyped form of a registered action, shared by entity and
// aggregate controllers.
type action struct {
	event  string
	filter Selector
	run    func(sc *scope, msg *bus.Message) error
}

type actionConfig struct {
	filter Selector
}

type ActionOption func(*actionConfig)

// Filter restricts the action to events whose labels match all sels. The
// event must implement Labeled.
func Filter(sels ...Selector) ActionOption {
	return func(c *actionConfig) { c.filter = AllOf(sels...) }
}

// newAction validates opts against E and builds the untyped action.
func newAction[E Event](owner string, opts []ActionOption) (*action, error) {
	var zero E

	cfg := actionConfig{}
	for _, o := range opts {
		o(&cfg)
	}

	a := &action{event: zero.EventName(), filter: cfg.filter}
	if a.event == "" {
		return nil, fmt.Errorf("%w: %T: empty EventName", ErrInvalidSchema, zero)
	}

	if _, ok := any(zero).(Labeled); a.filter != nil && !ok {
		return nil, fmt.Errorf("ee: %s: Filter on %T, which does not implement Labeled", owner, zero)
	}

	return a, nil
}

// decodeEvent decodes msg's payload as E. A failure is permanent: the same
// payload will never decode on redelivery.
func decodeEvent[E Event](env *Env, msg *bus.Message) (E, error) {
	var ev E
	if err := env.codec.Unmarshal(msg.Payload, &ev); err != nil {
		return ev, errPermanent{fmt.Errorf("ee: decode %s %s: %w", msg.Event, msg.ID, err)}
	}

	return ev, nil
}

// ActionCtx is what an EntityController action handler receives. The embedded
// Repo operates on the controller's own entity type.
type ActionCtx[S Entity, E Event] struct {
	context.Context //nolint:containedctx // handler contexts are request-scoped by design
	*Repo[S]

	sc    *scope
	event E
	meta  Meta
	reply func(context.Context, bus.Envelope) error
}

func (c *ActionCtx[S, E]) Event() E   { return c.event }
func (c *ActionCtx[S, E]) Meta() Meta { return c.meta }

// Entity loads the entity whose id is the event's Key. ErrNoKey if E does not
// implement Keyed.
func (c *ActionCtx[S, E]) Entity() (*S, error) {
	k, ok := any(c.event).(Keyed)
	if !ok {
		return nil, ErrNoKey
	}

	return c.Get(k.Key())
}

// Tx is the underlying store transaction, for backend-specific queries
// (e.g. postgres.PgxTx to run sqlc-generated code in the same transaction).
func (c *ActionCtx[S, E]) Tx() store.Tx { return c.sc.tx }

// Entities gives access to another entity type in the same transaction.
func (c *ActionCtx[S, E]) Entities[X Entity]() *Repo[X] { return repoOf[X](c.sc) }

// Emit queues an event in the transactional outbox; it is published only if
// the action commits.
func (c *ActionCtx[S, E]) Emit[T Event](ev T, opts ...SendOption) error { return c.sc.emit(ev, opts) }

// Reply answers a Request. ErrNotSupported when the delivery is an ordinary
// publish. The reply is not stored and is not published to other subscribers.
func (c *ActionCtx[S, E]) Reply(v any) error {
	return replyValue(c.sc.env, c.reply, c.Context, v)
}
