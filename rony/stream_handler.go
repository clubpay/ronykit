package rony

import (
	"reflect"
	"sync"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/rony/internal/stream"
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

func RegisterStream[IN, OUT Message, S State[A], A Action](
	setupCtx *SetupContext[S, A],
	h StreamHandler[S, A, IN, OUT],
	opt ...StreamOption,
) {
	var (
		in  IN
		out OUT
	)
	name := (*setupCtx.s).Name()
	s := setupCtx.s

	// we create the locker pointer to improve runtime performance, also
	// since Setup function guarantees that S is a pointer to a struct,
	sl, _ := any(*s).(sync.Locker)

	c := desc.NewContract().
		In(&in).
		Out(&out).
		SetName(reflect.TypeOf(h).Name()).
		SetHandler(
			func(ctx *kit.Context) {
				req := ctx.In().GetMsg().(*IN) //nolint:forcetypeassert
				err := h(newStreamCtx[S, A, OUT](ctx, s, sl), *req)
				if err != nil {
					ctx.Error(err)
				}
			},
		)

	cfg := stream.GenConfig(opt...)
	for _, s := range cfg.Selectors {
		c.AddNamedSelector(s.Name, s.Selector)
	}

	setupCtx.cfg.getService(name).AddContract(c)
}

func StreamDecoder(decoder DecoderFunc) StreamSelectorOption {
	return func(cfg *stream.SelectorConfig) {
		cfg.Decoder = fasthttp.DecoderFunc(decoder)
	}
}

/*

	StreamOption

*/

func RPC(predicate string, opt ...StreamSelectorOption) StreamOption {
	return func(cfg *stream.Config) {
		sCfg := stream.GenSelectorConfig(opt...)
		sCfg.Selector = fasthttp.RPC(predicate)

		cfg.Selectors = append(cfg.Selectors, sCfg)
	}
}
