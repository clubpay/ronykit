package di

import (
	"github.com/clubpay/ronykit/stub"
	"github.com/clubpay/ronykit/x/rkit"
	"github.com/clubpay/ronykit/x/telemetry/tracekit"
	"go.uber.org/fx"
)

// provideStub is a generic stub provider. This function panics if :
// 1. `Settings` is not a pointer to struct.
func provideStub[Settings, Stub any](
	moduleName, hostPortField string,
	constructor func(hostPort string, opt ...stub.Option) Stub,
	set Settings,
) Stub {
	return constructor(
		rkit.Coalesce(
			getField[string](set, "Services", hostPortField),
			"127.0.0.1:8586",
		),
		stub.WithTracePropagator(tracekit.B3(moduleName)),
	)
}

func StubProvider[Settings, IStub, Stub any](
	moduleName, hostPortField string,
	constructor func(hostPort string, opt ...stub.Option) Stub,
) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(set *Settings) Stub {
				return provideStub(moduleName, hostPortField, constructor, *set)
			},
			fx.As(new(IStub)),
		),
	)
}
