package ee

import (
	"context"
	"fmt"
	"iter"
	"reflect"

	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/x/rkit"
)

// StateController owns one state type S.
type StateController[S State] struct {
	env     *Env
	schema  *stateSchema
	actions map[string]*action
}

// NewStateController registers S under S.StateName().
func NewStateController[S State](env *Env) (*StateController[S], error) {
	var zero S

	t := reflect.TypeFor[S]()

	schema, err := parseStateSchema(zero.StateName(), t, env.codec)
	if err != nil {
		return nil, err
	}

	c := &StateController[S]{env: env, schema: schema, actions: map[string]*action{}}

	err = env.register(func() error {
		if _, ok := env.states[t]; ok {
			return fmt.Errorf("%w: state %s", ErrAlreadyRegistered, t)
		}

		if prev, ok := env.statesByName[schema.name]; ok {
			return fmt.Errorf("%w: state name %q is used by %s", ErrAlreadyRegistered, schema.name, prev.typ)
		}

		if err := env.addController(c.owner(), c); err != nil {
			return err
		}

		env.states[t] = schema
		env.statesByName[schema.name] = schema

		return nil
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *StateController[S]) Name() string { return c.schema.name }

// Get reads one state through a read-only transaction, which the store may
// serve from a replica.
func (c *StateController[S]) Get(ctx context.Context, id string) (S, error) {
	var out S

	err := c.env.view(ctx, nil, func(sc *scope) error {
		v, err := repoOf[S](sc).Get(id)
		if err == nil {
			out = *v
		}

		return err
	})

	return out, err
}

// Find streams the states matching q from one read-only transaction (possibly
// on a replica), which stays open while the caller iterates.
func (c *StateController[S]) Find(ctx context.Context, q Query) iter.Seq2[S, error] {
	return func(yield func(S, error) bool) {
		var zero S

		stopped := false

		err := c.env.view(ctx, nil, func(sc *scope) error {
			for v, err := range repoOf[S](sc).Find(q) {
				if err != nil {
					return err
				}

				if !yield(*v, nil) {
					stopped = true

					return nil
				}
			}

			return nil
		})
		if err != nil && !stopped {
			yield(zero, err)
		}
	}
}

func (c *StateController[S]) Create(ctx context.Context, v S) error {
	return c.env.unit(ctx, nil, false, func(sc *scope) error {
		return repoOf[S](sc).Create(&v)
	})
}

// Update loads the state, applies fn and saves it, retrying the whole
// read-modify-write on version conflict.
func (c *StateController[S]) Update(ctx context.Context, id string, fn func(*S) error) error {
	return c.env.unit(ctx, nil, true, func(sc *scope) error {
		r := repoOf[S](sc)

		v, err := r.Get(id)
		if err != nil {
			return err
		}

		if err := fn(v); err != nil {
			return err
		}

		return r.Save(v)
	})
}

func (c *StateController[S]) Delete(ctx context.Context, id string) error {
	return c.env.unit(ctx, nil, true, func(sc *scope) error {
		r := repoOf[S](sc)

		v, err := r.Get(id)
		if err != nil {
			return err
		}

		return r.Delete(v)
	})
}

// Action makes the controller the consumer of event E. Each event has exactly
// one consumer per Env. The handler runs in one transaction together with
// every Save and Emit it makes.
func (c *StateController[S]) Action[E Event](h func(*ActionCtx[S, E]) error, opts ...ActionOption) error {
	a, err := newAction[E](c.owner(), opts)
	if err != nil {
		return err
	}

	a.run = func(sc *scope, msg *bus.Message) error {
		ev, err := decodeEvent[E](c.env, msg)
		if err != nil {
			return err
		}

		return h(&ActionCtx[S, E]{
			Context: sc.ctx,
			Repo:    repoOf[S](sc),
			sc:      sc,
			event:   ev,
			meta:    metaOf(&msg.Envelope),
		})
	}

	return c.env.register(func() error {
		if err := c.env.claim(a.event, c.owner()); err != nil {
			return err
		}

		c.actions[a.event] = a

		return nil
	})
}

func (c *StateController[S]) owner() string { return "state:" + c.schema.name }

func (c *StateController[S]) subscribedEvents() []string { return eventNames(c.actions) }

func (c *StateController[S]) handle(ctx context.Context, msg bus.Message) error {
	return c.env.dispatch(ctx, c.owner(), c.actions, nil, msg)
}

func eventNames(actions map[string]*action) []string {
	return rkit.MapKeysToArray(actions)
}
