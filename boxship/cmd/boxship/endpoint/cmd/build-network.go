package cmd

import (
	"context"
	"time"

	"github.com/clubpay/ronykit/boxship"
	"github.com/clubpay/ronykit/boxship/pkg/log"
	"github.com/clubpay/ronykit/boxship/pkg/settings"
	"github.com/clubpay/ronykit/boxship/pkg/settings/viper"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func init() {
	BuildNetworkCmd.Flags().String(settings.Setup, "./setup", "")
}

var BuildNetworkCmd = &cobra.Command{
	Use:          "build-network",
	Short:        "builds the declared networks",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := fx.New(
			fx.NopLogger,
			fx.StartTimeout(time.Hour),
			fx.ErrorHook(errorHandler{}),
			fx.Supply(
				fx.Annotate(log.New(-1), fx.As(new(log.Logger))),
				cmd,
			),
			fx.Provide(viper.New, boxship.NewContext),
			fx.Decorate(settings.PrepareDefaults),
			fx.Invoke(buildNetwork(args...)),
		)
		err := app.Start(context.TODO())
		if err != nil {
			return err
		}

		return app.Stop(context.TODO())
	},
}

func buildNetwork(names ...string) func(lc fx.Lifecycle, bCtx *boxship.Context) {
	return func(lc fx.Lifecycle, bCtx *boxship.Context) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				if len(names) == 0 {
					bCtx.Log().Infof("creating all the networks defined in setup files")
				} else {
					bCtx.Log().Infof("creating networks: %v", names)
				}

				if len(names) == 0 {
					bCtx.ForEachContainer(
						func(desc boxship.ContainerDesc) {
							if desc.Disable {
								return
							}

							err := bCtx.BuildNetwork(desc.Name)
							if err != nil {
								bCtx.Log().Warnf("[%s] got error on creating network: %v", desc.Name, err)
							}
						},
					)

					return nil
				}

				return nil
			},
		})
	}
}
