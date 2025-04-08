package rony

import (
	"reflect"
	"runtime"
	"strings"
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

type RawUnaryHandler[
	S State[A], A Action,
	IN Message,
] func(ctx *UnaryCtx[S, A], in IN) (kit.RawMessage, error)

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
	sl, _ := any(*s).(sync.Locker) //nolint:errcheck

	handlers := make([]kit.HandlerFunc, 0, len(setupCtx.mw)+1)
	handlers = append(handlers, setupCtx.mw...)

	c := desc.NewContract()
	switch reflect.TypeOf(in) {
	default:
		handlers = append(handlers, CreateKitHandler[IN, OUT, S, A](h, s, sl, true))
		c.In(&in)
	case reflect.TypeOf(kit.RawMessage{}), reflect.TypeOf(kit.MultipartFormMessage{}):
		handlers = append(handlers, CreateKitHandler[IN, OUT, S, A](h, s, sl, false))
		c.In(in)
	}

	c.
		Out(&out).
		SetHandler(handlers...)

	if setupCtx.nodeSel != nil {
		c.SetCoordinator(setupCtx.nodeSel)
	}

	// Using reflection to get the name of the passed UnaryHandler function
	handlerName := ""
	hValue := reflect.ValueOf(h)
	if hValue.Kind() == reflect.Func {
		parts := strings.Split(runtime.FuncForPC(hValue.Pointer()).Name(), ".")
		handlerName = parts[len(parts)-1]
	}

	cfg := unary.GenConfig(opt...)
	for idx, s := range cfg.Selectors {
		if s.Name == "" {
			if idx == 0 {
				s.Name = utils.ToCamel(handlerName)
			} else {
				s.Name = utils.ToCamel(handlerName) + utils.IntToStr(idx+1)
			}
		}

		c.AddRoute(desc.Route(s.Name, s.Selector))
	}

	setupCtx.cfg.getService(setupCtx.name).AddContract(c)
}

func registerRawUnary[IN Message, S State[A], A Action](
	setupCtx *SetupContext[S, A],
	h RawUnaryHandler[S, A, IN],
	opt ...UnaryOption,
) {
	var (
		in  IN
		out kit.RawMessage
	)

	s := setupCtx.s
	// we create the locker pointer to improve runtime performance, also
	// since Setup function guarantees that S is a pointer to a struct,
	sl, _ := any(*s).(sync.Locker) //nolint:errcheck

	handlers := make([]kit.HandlerFunc, 0, len(setupCtx.mw)+1)
	handlers = append(handlers, setupCtx.mw...)

	c := desc.NewContract()
	switch reflect.TypeOf(in) {
	default:
		handlers = append(handlers, CreateRawKitHandler[IN, S, A](h, s, sl, true))
		c.In(&in)
	case reflect.TypeOf(kit.RawMessage{}), reflect.TypeOf(kit.MultipartFormMessage{}):
		handlers = append(handlers, CreateRawKitHandler[IN, S, A](h, s, sl, false))
		c.In(in)
	}

	c.
		Out(out).
		SetHandler(handlers...)

	if setupCtx.nodeSel != nil {
		c.SetCoordinator(setupCtx.nodeSel)
	}

	// Using reflection to get the name of the passed UnaryHandler function
	handlerName := ""
	hValue := reflect.ValueOf(h)
	if hValue.Kind() == reflect.Func {
		parts := strings.Split(runtime.FuncForPC(hValue.Pointer()).Name(), ".")
		handlerName = parts[len(parts)-1]
	}

	cfg := unary.GenConfig(opt...)
	for idx, s := range cfg.Selectors {
		if s.Name == "" {
			if idx == 0 {
				s.Name = utils.ToCamel(handlerName)
			} else {
				s.Name = utils.ToCamel(handlerName) + utils.IntToStr(idx+1)
			}
		}

		c.AddRoute(desc.Route(s.Name, s.Selector))
	}

	setupCtx.cfg.getService(setupCtx.name).AddContract(c)
}

func CreateKitHandler[IN, OUT Message, S State[A], A Action](
	h UnaryHandler[S, A, IN, OUT], s *S, sl sync.Locker,
	deRefIN bool,
) kit.HandlerFunc {
	return func(ctx *kit.Context) {
		var (
			err error
			req IN
			out *OUT
		)
		if deRefIN {
			req = *(ctx.In().GetMsg().(*IN)) //nolint:forcetypeassert,errcheck
		} else {
			req = ctx.In().GetMsg().(IN)
		}

		out, err = h(newUnaryCtx[S, A](ctx, s, sl), req)
		if err != nil {
			if e, ok := err.(errCode); ok {
				ctx.SetStatusCode(e.GetCode())
			}

			ctx.In().Reply().SetMsg(err).Send()

			return
		}

		ctx.Out().SetMsg(out).Send()
	}
}

func CreateRawKitHandler[IN Message, S State[A], A Action](
	h RawUnaryHandler[S, A, IN], s *S, sl sync.Locker,
	deRefIN bool,
) kit.HandlerFunc {
	return func(ctx *kit.Context) {
		var (
			err error
			req IN
			out kit.RawMessage
		)
		if deRefIN {
			req = *(ctx.In().GetMsg().(*IN)) //nolint:forcetypeassert,errcheck
		} else {
			req = ctx.In().GetMsg().(IN)
		}

		out, err = h(newUnaryCtx[S, A](ctx, s, sl), req)
		if err != nil {
			if e, ok := err.(errCode); ok {
				ctx.SetStatusCode(e.GetCode())
			}

			ctx.In().Reply().SetMsg(err).Send()

			return
		}

		ctx.Out().SetMsg(out).Send()
	}
}

/*
	UnarySelectorOption
*/

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
