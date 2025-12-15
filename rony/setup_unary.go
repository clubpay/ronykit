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

	cfg := genUnaryConfig(opt...)
	handlers = append(handlers, cfg.Middlewares...)

	c := desc.NewContract().
		SetInputHeader(cfg.Headers...)

	switch reflect.TypeOf(in) {
	default:
		handlers = append(handlers, CreateKitHandler[IN, OUT, S, A](h, s, sl, true))

		c.In(&in, cfg.InputMetaOptions...)
	case reflect.TypeFor[kit.RawMessage](), reflect.TypeFor[kit.MultipartFormMessage]():
		handlers = append(handlers, CreateKitHandler[IN, OUT, S, A](h, s, sl, false))

		c.In(in, cfg.InputMetaOptions...)
	}

	c.
		Out(&out, cfg.OutputMetaOptions...).
		SetDefaultError(&errs.Error{}).
		SetHandler(handlers...)

	if setupCtx.nodeSel != nil {
		c.SetCoordinator(setupCtx.nodeSel)
	}

	// Using reflection to get the name of the passed UnaryHandler function
	handlerName := ""

	hValue := reflect.ValueOf(h)
	if hValue.Kind() == reflect.Func {
		parts := strings.Split(runtime.FuncForPC(hValue.Pointer()).Name(), ".")
		handlerName = strings.TrimSuffix(parts[len(parts)-1], "Fm")
	}

	for idx, s := range cfg.Selectors {
		if s.Name == "" {
			if idx == 0 {
				s.Name = utils.ToCamel(handlerName)
			} else {
				s.Name = utils.ToCamel(handlerName) + utils.IntToStr(idx+1)
			}
		}

		route := desc.Route(s.Name, s.Selector)
		route.Deprecated = s.Deprecated
		c.AddRoute(route)
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

	cfg := genUnaryConfig(opt...)
	handlers = append(handlers, cfg.Middlewares...)

	c := desc.NewContract()

	switch reflect.TypeOf(in) {
	default:
		handlers = append(handlers, CreateRawKitHandler[IN, S, A](h, s, sl, true))

		c.In(&in, cfg.InputMetaOptions...)
	case reflect.TypeFor[kit.RawMessage](), reflect.TypeFor[kit.MultipartFormMessage]():
		handlers = append(handlers, CreateRawKitHandler[IN, S, A](h, s, sl, false))

		c.In(in, cfg.InputMetaOptions...)
	}

	c.
		Out(out, cfg.OutputMetaOptions...).
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
			err = errs.Convert(err)
			ctx.SetStatusCode(errs.HTTPStatus(err))

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
			req = ctx.In().GetMsg().(IN) //nolint:forcetypeassert,errcheck
		}

		out, err = h(newUnaryCtx[S, A](ctx, s, sl), req)
		if err != nil {
			ctx.SetStatusCode(errs.HTTPStatus(err))
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

type DecoderFunc = fasthttp.DecoderFunc

func UnaryDecoder(decoder DecoderFunc) UnarySelectorOption {
	return func(cfg *unarySelectorConfig) {
		cfg.Decoder = decoder
	}
}

func UnaryDeprecated(deprecated bool) UnarySelectorOption {
	return func(cfg *unarySelectorConfig) {
		cfg.Deprecated = deprecated
	}
}

/*
	UnaryOption
*/

func UnaryInputMeta(opt ...desc.MessageMetaOption) UnaryOption {
	return func(cfg *unaryConfig) {
		cfg.InputMetaOptions = opt
	}
}

func UnaryOutputMeta(opt ...desc.MessageMetaOption) UnaryOption {
	return func(cfg *unaryConfig) {
		cfg.OutputMetaOptions = opt
	}
}

var (
	OptionalHeader = desc.OptionalHeader
	RequiredHeader = desc.RequiredHeader
)

func UnaryHeader(hdr ...desc.Header) UnaryOption {
	return func(cfg *unaryConfig) {
		cfg.Headers = hdr
	}
}

func REST(method, path string, opt ...UnarySelectorOption) UnaryOption {
	return func(cfg *unaryConfig) {
		sCfg := genUnarySelectorConfig(opt...)
		sCfg.Selector = fasthttp.REST(method, path).SetDecoder(sCfg.Decoder)

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

func UnaryMiddleware(
	mw ...StatelessMiddleware,
) UnaryOption {
	return func(cfg *unaryConfig) {
		cfg.Middlewares = append(cfg.Middlewares, mw...)
	}
}

func UnaryMiddlewareFn(
	mw func() StatelessMiddleware,
) UnaryOption {
	return func(cfg *unaryConfig) {
		cfg.Middlewares = append(cfg.Middlewares, mw())
	}
}

type unaryConfig struct {
	Selectors         []unarySelectorConfig
	Middlewares       []StatelessMiddleware
	Headers           []desc.Header
	InputMetaOptions  []desc.MessageMetaOption
	OutputMetaOptions []desc.MessageMetaOption
}

func genUnaryConfig(opt ...UnaryOption) unaryConfig {
	cfg := unaryConfig{}

	for _, o := range opt {
		o(&cfg)
	}

	return cfg
}

type UnaryOption func(*unaryConfig)

type unarySelectorConfig struct {
	Decoder    fasthttp.DecoderFunc
	Name       string
	Deprecated bool
	Selector   kit.RESTRouteSelector
}

type UnarySelectorOption func(*unarySelectorConfig)

func genUnarySelectorConfig(opt ...UnarySelectorOption) unarySelectorConfig {
	cfg := unarySelectorConfig{}

	for _, o := range opt {
		o(&cfg)
	}

	return cfg
}
