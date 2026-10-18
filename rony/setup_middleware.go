package rony

import (
	"sync"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/x/rkit"
)

type Middleware[S State[A], A Action] interface {
	StatefulMiddleware[S, A] | StatelessMiddleware
}

type StatelessMiddleware = kit.HandlerFunc

type StatefulMiddleware[S State[A], A Action] = func(ctx *BaseCtx[S, A])

func statefulMiddlewareToKitHandler[S State[A], A Action](
	s *S,
	mw ...StatefulMiddleware[S, A],
) []kit.HandlerFunc {
	// we create the locker pointer to improve runtime performance, also
	// since Setup function guarantees that S is a pointer to a struct,
	sl := rkit.TryCast[sync.Locker](any(*s))

	return rkit.Map(mw, func(m StatefulMiddleware[S, A]) kit.HandlerFunc {
		return func(ctx *kit.Context) {
			m(newBaseCtx[S, A](ctx, s, sl))
		}
	})
}
