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
	BuildCmd.Flags().String(settings.Setup, "./setup", "")
	BuildCmd.Flags().Bool(settings.LogAll, true, "save build and runtime logs into separate files")
	BuildCmd.Flags().String(settings.WorkDir, "./_hdd", "we store all data in this working directory")
	BuildCmd.Flags().Bool(settings.Traefik, true, "enable traefik reverse proxy")
	BuildCmd.Flags().Bool(settings.ShallowClone, false, "shallow clone git repos")
	BuildCmd.Flags().Bool(settings.BuildKit, false, "use build-kit")
}

var BuildCmd = &cobra.Command{
	Use:          "build",
	Short:        "builds or pull docker images from repositories defined in setup files",
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
			fx.Invoke(buildImage(args...)),
		)
		err := app.Start(context.TODO())
		if err != nil {
			return err
		}

		return app.Stop(context.TODO())
	},
}

func buildImage(names ...string) func(lc fx.Lifecycle, bCtx *boxship.Context) {
	return func(lc fx.Lifecycle, bCtx *boxship.Context) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				bCtx.Log().Infof("building images: %v", names)

				if len(names) == 0 {
					bCtx.ForEachContainer(
						func(desc boxship.ContainerDesc) {
							if desc.Disable {
								return
							}

							err := bCtx.BuildImage(desc.Name)
							if err != nil {
								bCtx.Log().Warnf("[%s] got error on building: %v", desc.Name, err)
							}
						},
					)

					return nil
				}

				for _, name := range names {
					err := bCtx.BuildImage(name)
					if err != nil {
						return err
					}
				}

				return nil
			},
		})
	}
}
