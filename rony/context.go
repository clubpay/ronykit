package rony

import "github.com/clubpay/ronykit/kit"

type Context[A Action, SPtr StatePtr[S, A], S State[A]] struct {
	ctx *kit.Context
	s   SPtr
}

func newContext[A Action, SPtr StatePtr[S, A], S State[A]](ctx *kit.Context, s SPtr) *Context[A, SPtr, S] {
	return &Context[A, SPtr, S]{ctx: ctx, s: s}
}

func (c *Context[A, SPtr, S]) State() SPtr {
	return c.s
}
