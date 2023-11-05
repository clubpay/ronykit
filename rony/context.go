package rony

import (
	"context"
	"sync"

	"github.com/clubpay/ronykit/kit"
)

// BaseCtx is a base context object used by UnaryCtx and StreamCtx
// to provide common functionality.
type BaseCtx[S State[A], A Action] struct {
	ctx *kit.Context
	s   S
	sl  sync.Locker
}

func (c *BaseCtx[S, A]) State() S {
	return c.s
}

// ReduceState is a helper function to reduce state and call fn if it's not nil.
// If you need to reduce the state in an atomic fashion, then you should pass a
// function fn which is guaranteed to be called in a locked state.
// Although, it only works if S implements sync.Locker interface.
func (c *BaseCtx[S, A]) ReduceState(action A, fn func(s S, err error) error) (err error) {
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

func (c *BaseCtx[S, A]) Conn() kit.Conn {
	return c.ctx.Conn()
}

func (c *BaseCtx[S, A]) SetUserContext(userCtx context.Context) {
	c.ctx.SetUserContext(userCtx)
}

func (c *BaseCtx[S, A]) Context() context.Context {
	return c.ctx.Context()
}

func (c *BaseCtx[S, A]) Route() string {
	return c.ctx.Route()
}

func (c *BaseCtx[S, A]) Next() {
	c.ctx.Next()
}

// StopExecution stops the execution so the next middleware/handler won't be called.
func (c *BaseCtx[S, A]) StopExecution() {
	c.ctx.StopExecution()
}

/*
	UnaryCtx
*/

type UnaryCtx[S State[A], A Action] struct {
	BaseCtx[S, A]
}

func newUnaryCtx[S State[A], A Action](
	ctx *kit.Context, s *S, sl sync.Locker,
) *UnaryCtx[S, A] {
	return &UnaryCtx[S, A]{
		BaseCtx[S, A]{
			ctx: ctx,
			s:   *s,
			sl:  sl,
		},
	}
}

func (c *UnaryCtx[S, A]) RESTConn() (kit.RESTConn, bool) {
	if c.ctx.IsREST() {
		return c.ctx.RESTConn(), true
	}

	return nil, false
}

/*
	StreamCtx
*/

type StreamCtx[S State[A], A Action, M Message] struct {
	BaseCtx[S, A]
}

func newStreamCtx[S State[A], A Action, M Message](
	ctx *kit.Context, s *S, sl sync.Locker,
) *StreamCtx[S, A, M] {
	return &StreamCtx[S, A, M]{
		BaseCtx[S, A]{
			ctx: ctx,
			s:   *s,
			sl:  sl,
		},
	}
}

type PushOpt func(e *kit.Envelope)

func WithHdr(key, value string) PushOpt {
	return func(e *kit.Envelope) {
		e.SetHdr(key, value)
	}
}

func WithHdrMap(hdr map[string]string) PushOpt {
	return func(e *kit.Envelope) {
		e.SetHdrMap(hdr)
	}
}

func (c *StreamCtx[S, A, M]) Push(m M, opt ...PushOpt) S {
	c.PushTo(c.ctx.Conn(), m, opt...)

	return c.s
}

func (c *StreamCtx[S, A, M]) PushTo(conn kit.Conn, m M, opt ...PushOpt) {
	e := c.BaseCtx.ctx.
		OutTo(conn).
		SetMsg(m)
	for _, o := range opt {
		o(e)
	}

	e.Send()
}
