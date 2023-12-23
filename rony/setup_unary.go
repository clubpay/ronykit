package rony

import (
	"reflect"
	"sync"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/rony/internal/options/unary"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

// Exposing internal types
type (
	UnaryOption         = unary.Option
	UnarySelectorOption = unary.SelectorOption
)

type UnaryHandler[
	S State[A], A Action,
	IN, OUT Message,
] func(ctx *UnaryCtx[S, A], in IN) (*OUT, error)

func registerUnary[IN, OUT Message, S State[A], A Action](
	setupCtx *SetupContext[S, A],
	h UnaryHandler[S, A, IN, OUT],
	opt ...UnaryOption,
) {
	var (
		in  IN
		out OUT
	)

	s := setupCtx.s
	// we create the locker pointer to improve runtime performance, also
	// since Setup function guarantees that S is a pointer to a struct,
	sl, _ := any(*s).(sync.Locker)

	handlers := make([]kit.HandlerFunc, 0, len(setupCtx.mw)+1)
	handlers = append(handlers, setupCtx.mw...)
	handlers = append(handlers,
		func(ctx *kit.Context) {
			req := ctx.In().GetMsg().(*IN) //nolint:forcetypeassert
			out, err := h(newUnaryCtx[S, A](ctx, s, sl), *req)
			if err != nil {
				if e, ok := err.(errCode); ok {
					ctx.SetStatusCode(e.GetCode())
				}

				ctx.In().Reply().SetMsg(err).Send()

				return
			}

			ctx.Out().SetMsg(out).Send()
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

	cfg := unary.GenConfig(opt...)
	for _, s := range cfg.Selectors {
		c.AddNamedSelector(s.Name, s.Selector)
	}

	setupCtx.cfg.getService(setupCtx.name).AddContract(c)
}

/*
	UnarySelectorOption
*/

type DecoderFunc func(bag RESTParams, data []byte) (kit.Message, error)

func UnaryDecoder(decoder DecoderFunc) UnarySelectorOption {
	return func(cfg *unary.SelectorConfig) {
		cfg.Decoder = fasthttp.DecoderFunc(decoder)
	}
}

func UnaryName(name string) UnarySelectorOption {
	return func(cfg *unary.SelectorConfig) {
		cfg.Name = name
	}
}

/*
	UnaryOption
*/

func REST(method, path string, opt ...UnarySelectorOption) UnaryOption {
	return func(cfg *unary.Config) {
		sCfg := unary.GenSelectorConfig(opt...)
		sCfg.Selector = fasthttp.REST(method, path)

		cfg.Selectors = append(cfg.Selectors, sCfg)
	}
}

func ALL(path string, opt ...UnarySelectorOption) UnaryOption {
	return REST("*", path, opt...)
}

func GET(path string, opt ...UnarySelectorOption) UnaryOption {
	return REST("GET", path, opt...)
}

func POST(path string, opt ...UnarySelectorOption) UnaryOption {
	return REST("POST", path, opt...)
}

func PUT(path string, opt ...UnarySelectorOption) UnaryOption {
	return REST("PUT", path, opt...)
}

func DELETE(path string, opt ...UnarySelectorOption) UnaryOption {
	return REST("DELETE", path, opt...)
}

func PATCH(path string, opt ...UnarySelectorOption) UnaryOption {
	return REST("PATCH", path, opt...)
}

func HEAD(path string, opt ...UnarySelectorOption) UnaryOption {
	return REST("HEAD", path, opt...)
}

func OPTIONS(path string, opt ...UnarySelectorOption) UnaryOption {
	return REST("OPTIONS", path, opt...)
}
