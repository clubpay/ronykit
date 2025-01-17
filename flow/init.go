package flow

import (
	"github.com/clubpay/ronykit/kit/utils"
	"go.uber.org/fx"
)

type flowNS struct {
	namespace string
	taskQ     string
}

var flows = []flowNS{
	{namespace: "campaign", taskQ: "campaign"},
}

var Init = fx.Options(
	utils.Map(
		func(in flowNS) fx.Option {
			return fx.Provide(
				fx.Annotated{
					Name:   in.namespace + "Flow",
					Target: provideSDK(in.namespace, in.taskQ),
				})
		}, flows,
	)...,
)

func provideSDK(namespace, taskQ string) func(lc fx.Lifecycle, cfg Config) (*SDK, error) {
	return func(lc fx.Lifecycle, cfg Config) (*SDK, error) {
		return newSDK(lc, cfg, namespace, taskQ)
	}
}
