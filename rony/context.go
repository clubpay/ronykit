package rony

import (
	"context"
	"reflect"
	"sync"

	"github.com/clubpay/ronykit/kit"
)

type InitState[S State[A], A Action] func() S

func ToInitiateState[S State[A], A Action](s S) InitState[S, A] {
	return func() S {
		return s
	}
}

// SetupContext is a context object holds data until the Server
// starts.
// It is used internally to hold state and server configuration.
type SetupContext[S State[A], A Action] struct {
	s   *S
	cfg *serverConfig
}

// Setup is a helper function to set up server and services.
// Make sure S is a pointer to a struct, otherwise this function panics
func Setup[S State[A], A Action](srv *Server, stateFactory InitState[S, A]) *SetupContext[S, A] {
	state := stateFactory()
	if reflect.TypeOf(state).Kind() != reflect.Ptr {
		panic("state must be a pointer to a struct")
	}
	if reflect.TypeOf(state).Elem().Kind() != reflect.Struct {
		panic("state must be a pointer to a struct")
	}

	ctx := &SetupContext[S, A]{
		s:   &state,
		cfg: &srv.cfg,
	}

	return ctx
}

// baseCtx is a base context object which is used by UnaryCtx and StreamCtx
// to provide common functionality.
type baseCtx[S State[A], A Action] struct {
	ctx *kit.Context
	s   S
	sl  sync.Locker
}

func (c *baseCtx[S, A]) State() S {
	return c.s
}

// ReduceState is a helper function to reduce state and call fn if it's not nil.
// If you need to reduce the state in an atomic fashion, then you should pass a
// function fn which is guaranteed to be called in a locked state.
// Although, it only works if S implements sync.Locker interface.
func (c *baseCtx[S, A]) ReduceState(action A, fn func(s S, err error) error) (err error) {
	if c.sl != nil {
		c.sl.Lock()
		err = c.s.Reduce(action)
		if fn != nil {
			err = fn(c.s, err)
		}
		c.sl.Unlock()

		return err
	}

	err = c.s.Reduce(action)
	if fn != nil {
		err = fn(c.s, err)
	}

	return err
}

func (c *baseCtx[S, A]) Conn() kit.Conn {
	return c.ctx.Conn()
}

func (c *baseCtx[S, A]) SetUserContext(userCtx context.Context) {
	c.ctx.SetUserContext(userCtx)
}

func (c *baseCtx[S, A]) Context() context.Context {
	return c.ctx.Context()
}

func (c *baseCtx[S, A]) Route() string {
	return c.ctx.Route()
}

func (c *baseCtx[S, A]) Next() {
	c.ctx.Next()
}
