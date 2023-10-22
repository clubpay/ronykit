package rony

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
)

type SetupContext[A Action, S State[A], SPtr StatePtr[S, A]] struct {
	s   SPtr
	cfg *serverConfig
}

func Setup[A Action, SPtr StatePtr[S, A], S State[A]](srv *Server, f func() SPtr) *SetupContext[A, S, SPtr] {
	state := f()
	ctx := &SetupContext[A, S, SPtr]{
		s:   state,
		cfg: &srv.cfg,
	}

	return ctx
}

func RegisterHandler[A Action, SPtr StatePtr[S, A], S State[A], IN, OUT Message](
	setupCtx *SetupContext[A, S, SPtr],
	s RESTSelector,
	h Handler[A, S, SPtr, IN, OUT],
) {
	var (
		in  IN
		out OUT
	)
	setupCtx.cfg.getService(setupCtx.s.Name()).
		AddContract(
			desc.NewContract().
				In(&in).
				Out(&out).
				AddSelector(s).
				SetHandler(func(ctx *kit.Context) {
					in := ctx.In().GetMsg().(*IN) //nolint:forcetypeassert
					out, err := h(newContext[A, SPtr, S](ctx, setupCtx.s), *in)
					if err != nil {
						ctx.Out().SetMsg(err).Send()

						return
					}

					ctx.Out().SetMsg(out).Send()
				}),
		)
}
