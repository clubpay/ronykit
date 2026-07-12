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

type RelayHandler[S State[A], A Action] func(ctx *RelayCtx[S, A]) error

func registerRelay[S State[A], A Action](
	setupCtx *SetupContext[S, A],
	h RelayHandler[S, A],
	opt ...RelayOption,
) {
	s := setupCtx.s
	sl, _ := any(*s).(sync.Locker) //nolint:errcheck

	handlers := make([]kit.HandlerFunc, 0, len(setupCtx.mw)+1)
	handlers = append(handlers, setupCtx.mw...)

	cfg := genRelayConfig(opt...)
	applyRelayRESTBasePath(cfg.Selectors, setupCtx.basePath)
	handlers = append(handlers, cfg.Middlewares...)
	handlers = append(handlers, func(ctx *kit.Context) {
		err := h(newRelayCtx[S, A](ctx, s, sl))
		if err != nil {
			err = errs.Convert(err)
			ctx.SetStatusCode(errs.HTTPStatus(err))
			ctx.In().Reply().SetMsg(err).Send()
		}
	})

	c := desc.NewContract().
		In(kit.RawMessage(nil)).
		Out(kit.RawMessage(nil)).
		SetDefaultError(&errs.Error{}).
		SetHandler(handlers...)

	if setupCtx.nodeSel != nil {
		c.SetCoordinator(setupCtx.nodeSel)
	}

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

func RelayREST(method, path string, opt ...RelaySelectorOption) RelayOption {
	return func(cfg *relayConfig) {
		sCfg := genRelaySelectorConfig(opt...)
		sCfg.Selector = fasthttp.REST(method, path).SetDecoder(sCfg.Decoder)

		cfg.Selectors = append(cfg.Selectors, sCfg)
	}
}

func RelayALL(path string, opt ...RelaySelectorOption) RelayOption {
	return RelayREST("*", path, opt...)
}

func RelayGET(path string, opt ...RelaySelectorOption) RelayOption {
	return RelayREST("GET", path, opt...)
}

func RelayPOST(path string, opt ...RelaySelectorOption) RelayOption {
	return RelayREST("POST", path, opt...)
}

func RelayPUT(path string, opt ...RelaySelectorOption) RelayOption {
	return RelayREST("PUT", path, opt...)
}

func RelayDELETE(path string, opt ...RelaySelectorOption) RelayOption {
	return RelayREST("DELETE", path, opt...)
}

func RelayPATCH(path string, opt ...RelaySelectorOption) RelayOption {
	return RelayREST("PATCH", path, opt...)
}

func RelayHEAD(path string, opt ...RelaySelectorOption) RelayOption {
	return RelayREST("HEAD", path, opt...)
}

func RelayOPTIONS(path string, opt ...RelaySelectorOption) RelayOption {
	return RelayREST("OPTIONS", path, opt...)
}

func RelayMiddleware(mw ...StatelessMiddleware) RelayOption {
	return func(cfg *relayConfig) {
		cfg.Middlewares = append(cfg.Middlewares, mw...)
	}
}

func RelayDecoder(decoder fasthttp.DecoderFunc) RelaySelectorOption {
	return func(cfg *relaySelectorConfig) {
		cfg.Decoder = decoder
	}
}

func RelayName(name string) RelaySelectorOption {
	return func(cfg *relaySelectorConfig) {
		cfg.Name = name
	}
}

func RelayDeprecated(deprecated bool) RelaySelectorOption {
	return func(cfg *relaySelectorConfig) {
		cfg.Deprecated = deprecated
	}
}

type relayConfig struct {
	Selectors   []relaySelectorConfig
	Middlewares []StatelessMiddleware
}

type RelayOption func(*relayConfig)

type relaySelectorConfig struct {
	Decoder    fasthttp.DecoderFunc
	Name       string
	Deprecated bool
	Selector   kit.RESTRouteSelector
}

type RelaySelectorOption func(*relaySelectorConfig)

func genRelayConfig(opt ...RelayOption) relayConfig {
	cfg := relayConfig{}
	for _, o := range opt {
		o(&cfg)
	}

	return cfg
}

func genRelaySelectorConfig(opt ...RelaySelectorOption) relaySelectorConfig {
	cfg := relaySelectorConfig{}
	for _, o := range opt {
		o(&cfg)
	}

	return cfg
}

func applyRelayRESTBasePath(selectors []relaySelectorConfig, basePath string) {
	if basePath == "" {
		return
	}

	for idx := range selectors {
		sel, ok := selectors[idx].Selector.(fasthttp.Selector)
		if !ok || sel.Path == "" {
			continue
		}

		sel.Path = joinRESTPath(basePath, sel.Path)
		selectors[idx].Selector = sel
	}
}
