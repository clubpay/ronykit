package di

import (
	"fmt"
	"path/filepath"
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

type RegisterServiceParams struct {
	Kind        string
	Name        string
	InitFn      func(filename string, configPaths ...string)
	ModuleFn    func(opt ...fx.Option) fx.Option
	Middlewares []kit.HandlerFunc
}

func RegisterService[S any, SPtr ServicePtr[S]](
	params RegisterServiceParams,
) {
	m := _Services[params.Kind]
	if m == nil {
		m = map[string]ServiceOption{}
	}

	m[params.Name] = genModule[S, SPtr](params)
	_Services[params.Kind] = m
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

// configRoot is the base directory for bundled runtime config files. Bundled
// entrypoints (for example cmd/service) resolve per-kind paths under this root
// via ConfigSearchPath. Override with SetConfigRoot before fx starts.
var configRoot = "./config"

// SetConfigRoot sets the base directory used by ConfigSearchPath. Call this from
// a bundle entrypoint (for example after parsing a --config-dir flag) before
// starting the fx application.
func SetConfigRoot(root string) {
	if root == "" {
		configRoot = "./config"

		return
	}

	configRoot = root
}

// ConfigRoot returns the current bundled config base directory.
func ConfigRoot() string {
	return configRoot
}

var (
	ConfigFilename = func(name string) string {
		if idx := strings.LastIndex(name, "/"); idx != -1 {
			name = name[idx+1:]
		}

		return strings.ToLower(name) + ".local"
	}
	ConfigSearchPath = func(kind string) string {
		return filepath.Join(configRoot, kind)
	}
)

func genModule[
	S any, SPtr ServicePtr[S],
](
	params RegisterServiceParams,
) func(opt ...fx.Option) fx.Option {
	return func(opt ...fx.Option) fx.Option {
		return fx.Options(
			fx.Invoke(func() {
				if params.InitFn == nil {
					return
				}

				params.InitFn(ConfigFilename(params.Name), ConfigSearchPath(params.Kind))
			}),
			params.ModuleFn(opt...),
			fx.Invoke(
				fx.Annotate(
					func(srv *rony.Server, svc SPtr) {
						setupRony(srv, rkit.ToCamel(params.Name), svc.Desc(), params.Middlewares...)
					},
					fx.ParamTags(fmt.Sprintf("name:%q", params.Kind)),
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
