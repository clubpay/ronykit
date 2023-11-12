package rony

import (
	"sync"

	"github.com/clubpay/ronykit/kit"
)

type Middleware[S State[A], A Action] interface {
	StatefulMiddleware[S, A] | StatelessMiddleware
}

type StatelessMiddleware = kit.HandlerFunc

type StatefulMiddleware[S State[A], A Action] func(ctx *BaseCtx[S, A])

func registerStatelessMiddleware[S State[A], A Action](
	setupCtx *SetupContext[S, A],
	h ...StatelessMiddleware,
) {
	setupCtx.mw = append(setupCtx.mw, h...)
}

func registerStatefulMiddleware[S State[A], A Action](
	setupCtx *SetupContext[S, A],
	h ...StatefulMiddleware[S, A],
) {
	s := setupCtx.s
	// we create the locker pointer to improve runtime performance, also
	// since Setup function guarantees that S is a pointer to a struct,
	sl, _ := any(*s).(sync.Locker)

	for _, m := range h {
		setupCtx.mw = append(
			setupCtx.mw,
			func(ctx *kit.Context) {
				m(newBaseCtx[S, A](ctx, s, sl))
			},
		)
	}
}
