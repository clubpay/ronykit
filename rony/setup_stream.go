package rony

import (
	"reflect"
	"sync"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/rony/internal/options/stream"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

// Exposing internal types
type (
	StreamOption         = stream.Option
	StreamSelectorOption = stream.SelectorOption
)

type StreamHandler[
	S State[A], A Action,
	IN, OUT Message,
] func(ctx *StreamCtx[S, A, OUT], in IN) error

func registerStream[IN, OUT Message, S State[A], A Action](
	setupCtx *SetupContext[S, A],
	h StreamHandler[S, A, IN, OUT],
	opt ...StreamOption,
) {
	var (
		in  IN
		out OUT
	)

	s := setupCtx.s
	// we create the locker pointer to improve runtime performance, also
	// since Setup function guarantees that S is a pointer to a struct,
	sl, _ := any(*s).(sync.Locker) //nolint:errcheck

	handlers := make([]kit.HandlerFunc, 0, len(setupCtx.mw)+1)
	handlers = append(handlers, setupCtx.mw...)
	handlers = append(handlers,
		func(ctx *kit.Context) {
			req := ctx.In().GetMsg().(*IN) //nolint:forcetypeassert,errcheck
			err := h(newStreamCtx[S, A, OUT](ctx, s, sl), *req)
			if err != nil {
				ctx.Error(err)
			}
		},
	)

	c := desc.NewContract().
		In(&in).
		Out(&out).
		SetName(reflect.TypeOf(h).Name()).
		SetHandler(handlers...)

	if setupCtx.nodeSel != nil {
		c.SetCoordinator(setupCtx.nodeSel)
	}

	cfg := stream.GenConfig(opt...)
	for _, s := range cfg.Selectors {
		c.AddNamedSelector(s.Name, s.Selector)
	}

	setupCtx.cfg.getService(setupCtx.name).AddContract(c)
}

/*
	StreamOption
*/

// RPC is a StreamOption to set up RPC handler.
func RPC(predicate string, opt ...StreamSelectorOption) StreamOption {
	return func(cfg *stream.Config) {
		sCfg := stream.GenSelectorConfig(opt...)
		sCfg.Selector = fasthttp.RPC(predicate)

		cfg.Selectors = append(cfg.Selectors, sCfg)
	}
}
