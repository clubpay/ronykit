package ee

import (
	"context"
	"fmt"
	"iter"
	"reflect"

	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/x/rkit"
)

// EntityController owns one entity type S.
type EntityController[S Entity] struct {
	env     *Env
	schema  *entitySchema
	actions map[string]*action
}

// NewEntityController registers S under S.EntityName().
func NewEntityController[S Entity](env *Env) (*EntityController[S], error) {
	var zero S

	t := reflect.TypeFor[S]()

	schema, err := parseEntitySchema(zero.EntityName(), t, env.codec)
	if err != nil {
		return nil, err
	}

	c := &EntityController[S]{env: env, schema: schema, actions: map[string]*action{}}

	err = env.register(func() error {
		if _, ok := env.entities[t]; ok {
			return fmt.Errorf("%w: entity %s", ErrAlreadyRegistered, t)
		}

		if prev, ok := env.entitiesByName[schema.name]; ok {
			return fmt.Errorf("%w: entity name %q is used by %s", ErrAlreadyRegistered, schema.name, prev.typ)
		}

		if err := env.addController(c.owner(), c); err != nil {
			return err
		}

		env.entities[t] = schema
		env.entitiesByName[schema.name] = schema

		return nil
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *EntityController[S]) Name() string { return c.schema.name }

// Get reads one state through a read-only transaction, which the store may
// serve from a replica.
func (c *EntityController[S]) Get(ctx context.Context, id string) (S, error) {
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
func (c *EntityController[S]) Find(ctx context.Context, q Query) iter.Seq2[S, error] {
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

func (c *EntityController[S]) Create(ctx context.Context, v S) error {
	return c.env.unit(ctx, nil, false, func(sc *scope) error {
		return repoOf[S](sc).Create(&v)
	})
}

// Update loads the state, applies fn and saves it, retrying the whole
// read-modify-write on version conflict.
func (c *EntityController[S]) Update(ctx context.Context, id string, fn func(*S) error) error {
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

func (c *EntityController[S]) Delete(ctx context.Context, id string) error {
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
func (c *EntityController[S]) Action[E Event](h func(*ActionCtx[S, E]) error, opts ...ActionOption) error {
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
			reply:   msg.Reply,
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

func (c *EntityController[S]) owner() string { return "entity:" + c.schema.name }

func (c *EntityController[S]) subscribedEvents() []string { return eventNames(c.actions) }

func (c *EntityController[S]) handle(ctx context.Context, msg bus.Message) error {
	return c.env.dispatch(ctx, c.owner(), c.actions, nil, msg)
}

func eventNames(actions map[string]*action) []string {
	return rkit.MapKeysToArray(actions)
}
