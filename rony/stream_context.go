package rony

import (
	"sync"

	"github.com/clubpay/ronykit/kit"
)

type StreamCtx[S State[A], A Action, M Message] struct {
	baseCtx[S, A]
}

func newStreamCtx[S State[A], A Action, M Message](
	ctx *kit.Context, s *S, sl sync.Locker,
) *StreamCtx[S, A, M] {
	return &StreamCtx[S, A, M]{
		baseCtx[S, A]{
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
	e := c.baseCtx.ctx.
		OutTo(conn).
		SetMsg(m)
	for _, o := range opt {
		o(e)
	}

	e.Send()
}
