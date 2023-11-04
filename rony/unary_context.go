package rony

import (
	"sync"

	"github.com/clubpay/ronykit/kit"
)

type UnaryCtx[S State[A], A Action] struct {
	baseCtx[S, A]
}

func newUnaryCtx[S State[A], A Action](
	ctx *kit.Context, s *S, sl sync.Locker,
) *UnaryCtx[S, A] {
	return &UnaryCtx[S, A]{
		baseCtx[S, A]{
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
