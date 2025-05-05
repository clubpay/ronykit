package rony

import (
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/rony/errs"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
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
	opt ...UnaryOption[S, A],
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

	cfg := genUnaryConfig(setupCtx.s, opt...)
	handlers = append(handlers, cfg.Middlewares...)

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
	opt ...UnaryOption[S, A],
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

	cfg := genUnaryConfig(setupCtx.s, opt...)
	handlers = append(handlers, cfg.Middlewares...)

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
			req = ctx.In().GetMsg().(IN) //nolint:forcetypeassert,errcheck
		}

		out, err = h(newUnaryCtx[S, A](ctx, s, sl), req)
		if err != nil {
			if e, ok := err.(errCode); ok {
				ctx.SetStatusCode(e.GetCode())
			}

			ctx.In().Reply().SetMsg(errs.Convert(err)).Send()

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
			req = ctx.In().GetMsg().(IN) //nolint:forcetypeassert,errcheck
		}

		out, err = h(newUnaryCtx[S, A](ctx, s, sl), req)
		if err != nil {
			if e, ok := err.(errCode); ok {
				ctx.SetStatusCode(e.GetCode())
			}

			ctx.In().Reply().SetMsg(errs.Convert(err)).Send()

			return
		}

		ctx.Out().SetMsg(out).Send()
	}
}

/*
	UnarySelectorOption
*/

func UnaryName(name string) UnarySelectorOption {
	return func(cfg *unarySelectorConfig) {
		cfg.Name = name
	}
}

/*
	UnaryOption
*/

func REST[S State[A], A Action](method, path string, opt ...UnarySelectorOption) UnaryOption[S, A] {
	return func(cfg *unaryConfig[S, A]) {
		sCfg := genUnarySelectorConfig(opt...)
		sCfg.Selector = fasthttp.REST(method, path)

		cfg.Selectors = append(cfg.Selectors, sCfg)
	}
}

func ALL[S State[A], A Action](path string, opt ...UnarySelectorOption) UnaryOption[S, A] {
	return REST[S, A]("*", path, opt...)
}

func GET[S State[A], A Action](path string, opt ...UnarySelectorOption) UnaryOption[S, A] {
	return REST[S, A]("GET", path, opt...)
}

func POST[S State[A], A Action](path string, opt ...UnarySelectorOption) UnaryOption[S, A] {
	return REST[S, A]("POST", path, opt...)
}

func PUT[S State[A], A Action](path string, opt ...UnarySelectorOption) UnaryOption[S, A] {
	return REST[S, A]("PUT", path, opt...)
}

func DELETE[S State[A], A Action](path string, opt ...UnarySelectorOption) UnaryOption[S, A] {
	return REST[S, A]("DELETE", path, opt...)
}

func PATCH[S State[A], A Action](path string, opt ...UnarySelectorOption) UnaryOption[S, A] {
	return REST[S, A]("PATCH", path, opt...)
}

func HEAD[S State[A], A Action](path string, opt ...UnarySelectorOption) UnaryOption[S, A] {
	return REST[S, A]("HEAD", path, opt...)
}

func OPTIONS[S State[A], A Action](path string, opt ...UnarySelectorOption) UnaryOption[S, A] {
	return REST[S, A]("OPTIONS", path, opt...)
}

func UnaryMiddleware[S State[A], A Action, M Middleware[S, A]](
	m ...M,
) UnaryOption[S, A] {
	return func(cfg *unaryConfig[S, A]) {
		for _, m := range m {
			switch mw := any(m).(type) {
			case StatefulMiddleware[S, A]:
				cfg.Middlewares = append(cfg.Middlewares, statefulMiddlewareToKitHandler[S, A](cfg.s, mw)...)

			case StatelessMiddleware:
				cfg.Middlewares = append(cfg.Middlewares, mw)
			}
		}
	}
}

type unaryConfig[S State[A], A Action] struct {
	s           *S
	Selectors   []unarySelectorConfig
	Middlewares []kit.HandlerFunc
}

func genUnaryConfig[S State[A], A Action](s *S, opt ...UnaryOption[S, A]) unaryConfig[S, A] {
	cfg := unaryConfig[S, A]{}

	for _, o := range opt {
		o(&cfg)
	}

	return cfg
}

type UnaryOption[S State[A], A Action] func(*unaryConfig[S, A])

type unarySelectorConfig struct {
	Decoder  fasthttp.DecoderFunc
	Name     string
	Selector kit.RouteSelector
}

type UnarySelectorOption func(*unarySelectorConfig)

func genUnarySelectorConfig(opt ...UnarySelectorOption) unarySelectorConfig {
	cfg := unarySelectorConfig{}

	for _, o := range opt {
		o(&cfg)
	}

	return cfg
}
