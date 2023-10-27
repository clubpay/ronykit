package rony

import (
	"reflect"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
)

type SetupContext[A Action, S State[A]] struct {
	s   *S
	cfg *serverConfig
}

// Setup is a helper function to set up server and services.
// Make sure S is a pointer to a struct, otherwise this function panics
func Setup[A Action, S State[A]](srv *Server, f func() S) *SetupContext[A, S] {
	state := f()
	if reflect.TypeOf(state).Kind() != reflect.Ptr {
		panic("state must be a pointer to a struct")
	}
	if reflect.TypeOf(state).Elem().Kind() != reflect.Struct {
		panic("state must be a pointer to a struct")
	}

	ctx := &SetupContext[A, S]{
		s:   &state,
		cfg: &srv.cfg,
	}

	return ctx
}

func RegisterHandler[A Action, S State[A], IN, OUT Message](
	setupCtx *SetupContext[A, S],
	s RESTSelector,
	h Handler[A, S, IN, OUT],
) {
	var (
		in  IN
		out OUT
	)
	name := (*setupCtx.s).Name()
	setupCtx.cfg.getService(name).
		AddContract(
			desc.NewContract().
				In(&in).
				Out(&out).
				AddSelector(s).
				SetName(reflect.TypeOf(h).Name()).
				SetHandler(func(ctx *kit.Context) {
					in := ctx.In().GetMsg().(*IN) //nolint:forcetypeassert
					out, err := h(newContext[A, S](ctx, setupCtx.s), *in)
					if err != nil {
						ctx.Out().SetMsg(err).Send()

						return
					}

					ctx.Out().SetMsg(out).Send()
				}),
		)
}
