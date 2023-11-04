package rony

import (
	"context"
	"sync"

	"github.com/clubpay/ronykit/kit"
)

type Context[S State[A], A Action] struct {
	ctx *kit.Context
	s   S
	sl  sync.Locker
}

func newContext[S State[A], A Action](
	ctx *kit.Context, s *S, sl sync.Locker,
) *Context[S, A] {
	return &Context[S, A]{ctx: ctx, s: *s, sl: sl}
}

func (c *Context[S, A]) State() S {
	return c.s
}

// ReduceState is a helper function to reduce state and call fn if it's not nil.
// If you need to reduce the state in an atomic fashion, then you should pass a
// function fn which is guaranteed to be called in a locked state.
// Although, it only works if S implements sync.Locker interface.
func (c *Context[S, A]) ReduceState(action A, fn func(s S)) {
	if c.sl != nil {
		c.sl.Lock()
		c.s.Reduce(action)
		if fn != nil {
			fn(c.s)
		}
		c.sl.Unlock()

		return
	}

	c.s.Reduce(action)
	if fn != nil {
		fn(c.s)
	}

	return
}

func (c *Context[S, A]) Conn() kit.Conn {
	return c.ctx.Conn()
}

func (c *Context[S, A]) SetUserContext(userCtx context.Context) {
	c.ctx.SetUserContext(userCtx)
}

func (c *Context[S, A]) Context() context.Context {
	return c.ctx.Context()
}

func (c *Context[S, A]) Route() string {
	return c.ctx.Route()
}

func (c *Context[S, A]) Next() {
	c.ctx.Next()
}
