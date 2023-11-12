package rony

import (
	"context"
	"sync"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
)

// BaseCtx is a base context object used by UnaryCtx and StreamCtx
// to provide common functionality.
type BaseCtx[S State[A], A Action] struct {
	ctx *kit.Context
	s   S
	sl  sync.Locker
}

func newBaseCtx[S State[A], A Action](
	ctx *kit.Context, s *S, sl sync.Locker,
) *BaseCtx[S, A] {
	return &BaseCtx[S, A]{
		ctx: ctx,
		s:   *s,
		sl:  sl,
	}
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

// KitCtx returns the underlying kit.Context. This is useful when we need to
// pass the context into inner layers of our application when we need to have
// access to more generic functionalities.
func (c *BaseCtx[S, A]) KitCtx() *kit.Context {
	return c.ctx
}

func (c *BaseCtx[S, A]) Set(key string, value any) {
	c.ctx.Set(key, value)
}

func (c *BaseCtx[S, A]) Get(key string) any {
	return c.ctx.Get(key)
}

func (c *BaseCtx[S, A]) Walk(f func(key string, val any) bool) {
	c.ctx.Walk(f)
}

func (c *BaseCtx[S, A]) Exists(key string) bool {
	return c.ctx.Exists(key)
}

func (c *BaseCtx[S, A]) GetInputHeader(key string) string {
	return c.ctx.In().GetHdr(key)
}

// GetInHdr is shorthand for GetInputHeader
func (c *BaseCtx[S, A]) GetInHdr(key string) string {
	return c.GetInputHeader(key)
}

func (c *BaseCtx[S, A]) WalkInputHeader(f func(key string, val string) bool) {
	c.ctx.In().WalkHdr(f)
}

// WalkInHdr is shorthand for WalkInputHeader
func (c *BaseCtx[S, A]) WalkInHdr(f func(key string, val string) bool) {
	c.WalkInputHeader(f)
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
		BaseCtx: utils.PtrVal(newBaseCtx[S, A](ctx, s, sl)),
	}
}

// RESTConn returns the underlying RESTConn if the connection is RESTConn.
func (c *UnaryCtx[S, A]) RESTConn() (kit.RESTConn, bool) {
	if c.ctx.IsREST() {
		return c.ctx.RESTConn(), true
	}

	return nil, false
}

// SetOutputHeader sets the output header
func (c *UnaryCtx[S, A]) SetOutputHeader(key, value string) {
	c.ctx.PresetHdr(key, value)
}

// SetOutHdr is shorthand for SetOutputHeader
func (c *UnaryCtx[S, A]) SetOutHdr(key, value string) {
	c.SetOutputHeader(key, value)
}

// SetOutputHeaderMap sets the output header map
func (c *UnaryCtx[S, A]) SetOutputHeaderMap(kv map[string]string) {
	c.ctx.PresetHdrMap(kv)
}

// SetOutHdrMap is shorthand for SetOutputHeaderMap
func (c *UnaryCtx[S, A]) SetOutHdrMap(kv map[string]string) {
	c.SetOutputHeaderMap(kv)
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
		BaseCtx: utils.PtrVal(newBaseCtx[S, A](ctx, s, sl)),
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
