package di

import (
	"fmt"
	"slices"
	"strings"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/rony"
	"github.com/clubpay/ronykit/x/rkit"
	"go.uber.org/fx"
)

type (
	ServiceOption     = func(opt ...fx.Option) fx.Option
	ServicePtr[S any] interface {
		*S
		Desc() rony.SetupOption[rony.EMPTY, rony.NOP]
	}
)

var _Services = map[string]map[string]ServiceOption{}

func RegisterService[S any, SPtr ServicePtr[S]](
	kind, name string,
	initFn func(filename string, configPaths ...string),
	moduleFn func(opt ...fx.Option) fx.Option,
	mw ...kit.HandlerFunc,
) {
	m := _Services[kind]
	if m == nil {
		m = map[string]ServiceOption{}
	}
	m[name] = genModule[S, SPtr](kind, name, initFn, moduleFn, mw...)
	_Services[kind] = m
}

var _Middlewares []kit.HandlerFunc

func RegisterMiddleware(mw ...kit.HandlerFunc) {
	_Middlewares = append(_Middlewares, mw...)
}

func AllServices() []ServiceOption {
	var opts []ServiceOption
	for k := range _Services {
		for _, opt := range _Services[k] {
			opts = append(opts, opt)
		}
	}

	return opts
}

func GetService(kind, name string) func(opt ...fx.Option) fx.Option {
	m := _Services[kind]
	if m == nil {
		return nil
	}

	return m[name]
}

func GetServiceByKind(kind string) map[string]ServiceOption {
	return _Services[kind]
}

var (
	ConfigFilename = func(name string) string {
		if idx := strings.LastIndex(name, "/"); idx != -1 {
			name = name[idx+1:]
		}

		return strings.ToLower(name) + ".local"
	}
	ConfigSearchPath = func(kind string) string { return fmt.Sprintf("./config/%s", kind) }
)

func genModule[
	S any, SPtr ServicePtr[S],
](
	kind, name string,
	initFn func(filename string, configPaths ...string),
	moduleFn func(opt ...fx.Option) fx.Option,
	mw ...kit.HandlerFunc,
) func(opt ...fx.Option) fx.Option {
	return func(opt ...fx.Option) fx.Option {
		return fx.Options(
			fx.Invoke(func() {
				if initFn == nil {
					return
				}

				initFn(ConfigFilename(name), ConfigSearchPath(kind))
			}),
			moduleFn(opt...),
			fx.Invoke(
				fx.Annotate(
					func(srv *rony.Server, svc SPtr) {
						setupRony(srv, rkit.ToCamel(name), svc.Desc(), mw...)
					},
					fx.ParamTags(fmt.Sprintf("name:%q", kind)),
				),
			),
		)
	}
}

func setupRony(
	srv *rony.Server,
	name string,
	option rony.SetupOption[rony.EMPTY, rony.NOP],
	mw ...kit.HandlerFunc,
) {
	rony.Setup(
		srv, name, rony.EmptyState(),
		rony.WithMiddleware[rony.EMPTY](append(slices.Clone(_Middlewares), mw...)...),
		option,
	)
}
