package di

import (
	"fmt"
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

var _Services = map[string]ServiceOption{}

func RegisterService[S any, SPtr ServicePtr[S]](
	kind, name string,
	initFn func(filename string, configPaths ...string),
	moduleFn func(opt ...fx.Option) fx.Option,
) {
	_Services[kind+"/"+name] = genModule[S, SPtr](
		kind, name, initFn, moduleFn,
	)
}

var _Middlewares []kit.HandlerFunc

func RegisterMiddleware(mw ...kit.HandlerFunc) {
	_Middlewares = append(_Middlewares, mw...)
}

func AllServices() []ServiceOption {
	var opts []ServiceOption
	for _, opt := range _Services {
		opts = append(opts, opt)
	}

	return opts
}

func GetService(kind, name string) func(opt ...fx.Option) fx.Option {
	return _Services[kind+"/"+name]
}

func genModule[
	S any, SPtr ServicePtr[S],
](
	typ, name string,
	initFn func(filename string, configPaths ...string),
	moduleFn func(opt ...fx.Option) fx.Option,
) func(opt ...fx.Option) fx.Option {
	return func(opt ...fx.Option) fx.Option {
		return fx.Options(
			fx.Invoke(func() {
				initFn(
					strings.ToLower(name)+".local",
					fmt.Sprintf("./configs/%ss", typ),
				)
			}),
			moduleFn(opt...),
			fx.Invoke(
				fx.Annotate(
					func(srv *rony.Server, svc SPtr) {
						setupRony(srv, rkit.ToCamel(name), svc.Desc())
					},
					fx.ParamTags(fmt.Sprintf("name:%q", typ)),
				),
			),
		)
	}
}

func setupRony(srv *rony.Server, name string, option rony.SetupOption[rony.EMPTY, rony.NOP]) {
	rony.Setup(
		srv, name, rony.EmptyState(),
		rony.WithMiddleware[rony.EMPTY](_Middlewares...),
		option,
	)
}
