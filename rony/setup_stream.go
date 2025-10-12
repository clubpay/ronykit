package rony

import (
	"reflect"
	"sync"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/rony/errs"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
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
				ctx.Error(errs.Convert(err))
			}
		},
	)

	c := desc.NewContract().
		SetName(reflect.TypeOf(h).Name()).
		SetHandler(handlers...)

	if setupCtx.nodeSel != nil {
		c.SetCoordinator(setupCtx.nodeSel)
	}

	cfg := genStreamConfig(opt...)
	for _, s := range cfg.Selectors {
		c.AddRoute(desc.Route(s.Name, s.Selector))
	}

	c.In(&in, cfg.InputMetaOptions...).
		Out(&out, cfg.OutputMetaOptions...)

	setupCtx.cfg.getService(setupCtx.name).AddContract(c)
}

/*
	StreamOption
*/

func StreamInputMeta(opt ...desc.MessageMetaOption) StreamOption {
	return func(cfg *streamConfig) {
		cfg.InputMetaOptions = opt
	}
}

func StreamOutputMeta(opt ...desc.MessageMetaOption) StreamOption {
	return func(cfg *streamConfig) {
		cfg.OutputMetaOptions = opt
	}
}

// RPC is a StreamOption to set up an RPC handler.
func RPC(predicate string, opt ...StreamSelectorOption) StreamOption {
	return func(cfg *streamConfig) {
		sCfg := genStreamSelectorConfig(opt...)
		sCfg.Selector = fasthttp.RPC(predicate)

		cfg.Selectors = append(cfg.Selectors, sCfg)
	}
}

type streamConfig struct {
	Selectors         []streamSelectorConfig
	InputMetaOptions  []desc.MessageMetaOption
	OutputMetaOptions []desc.MessageMetaOption
}

func genStreamConfig(opt ...StreamOption) streamConfig {
	cfg := streamConfig{}

	for _, o := range opt {
		o(&cfg)
	}

	return cfg
}

type StreamOption func(*streamConfig)

type streamSelectorConfig struct {
	Decoder  fasthttp.DecoderFunc
	Name     string
	Selector kit.RPCRouteSelector
}

type StreamSelectorOption func(*streamSelectorConfig)

func genStreamSelectorConfig(opt ...StreamSelectorOption) streamSelectorConfig {
	cfg := streamSelectorConfig{}

	for _, o := range opt {
		o(&cfg)
	}

	return cfg
}
