package rony

import (
	"github.com/clubpay/ronykit/kit"
)

type StatelessMiddleware = kit.HandlerFunc

type Middleware[S State[A], A Action] func(ctx *BaseCtx[S, A]) error

func registerStatelessMiddleware[S State[A], A Action](
	setupCtx *SetupContext[S, A],
	h ...StatelessMiddleware,
) {
	setupCtx.cfg.mw = append(setupCtx.cfg.mw, h...)
}
