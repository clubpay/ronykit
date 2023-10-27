package rony

import "github.com/clubpay/ronykit/kit"

type Context[A Action, S State[A]] struct {
	ctx *kit.Context
	s   *S
}

func newContext[A Action, S State[A]](ctx *kit.Context, s *S) *Context[A, S] {
	return &Context[A, S]{ctx: ctx, s: s}
}

func (c *Context[A, S]) State() S {
	return *c.s
}
