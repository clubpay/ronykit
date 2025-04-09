package cmd

import (
	"context"
	"time"

	"github.com/clubpay/ronykit/boxship"
	"github.com/clubpay/ronykit/boxship/pkg/container/preset"
	"github.com/clubpay/ronykit/boxship/pkg/log"
	"github.com/clubpay/ronykit/boxship/pkg/settings"
	"github.com/clubpay/ronykit/boxship/pkg/settings/viper"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func init() {
	RunCmd.Flags().Bool(settings.LogAll, true, "save build and runtime logs into separate files")
	RunCmd.Flags().String(settings.WorkDir, "./_hdd", "we store all data in this working directory")
	RunCmd.Flags().Bool(settings.Traefik, true, "enable traefik reverse proxy")
	RunCmd.Flags().Bool(settings.ShallowClone, false, "shallow clone git repos")
}

var RunCmd = &cobra.Command{
	Use:          "run",
	Short:        "run the containers",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fx.New(
			// fx.NopLogger,
			fx.ErrorHook(errorHandler{}),
			fx.Supply(
				fx.Annotate(log.New(-1), fx.As(new(log.Logger))),
				cmd,
			),
			fx.Provide(viper.New, boxship.NewContext),
			fx.StartTimeout(time.Hour),
			fx.Decorate(settings.PrepareDefaults),
			fx.Invoke(runContainers(args...)),
		).Run()

		return nil
	},
}

func runContainers(names ...string) func(lc fx.Lifecycle, bCtx *boxship.Context) {
	return func(lc fx.Lifecycle, bCtx *boxship.Context) {
		var ids []string

		startFunc := genStartFunc(bCtx, &ids)

		var onStart func(ctx context.Context) error
		if len(names) == 0 {
			onStart = func(ctx context.Context) error {
				bCtx.ForEachContainer(startFunc)

				return nil
			}
		} else {
			onStart = func(ctx context.Context) error {
				for _, name := range names {
					id, err := bCtx.RunContainer(name)
					if err != nil {
						bCtx.Log().Warnf("[%s] got error on running: %v", name, err)

						return err
					}
					ids = append(ids, id)
				}

				return nil
			}
		}

		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				bCtx.Log().Infof("running containers: %v", names)
				err := bCtx.CreateNetwork(settings.TraefikNetwork)
				if err != nil {
					bCtx.Log().Warnf("got error on creating network[%s]: %v", settings.TraefikNetwork, err)
				}
				bCtx.RegisterContainerDesc(preset.TraefikX(bCtx.Settings()))

				return onStart(ctx)
			},
			OnStop: func(ctx context.Context) error {
				for n := len(ids) - 1; n >= 0; n-- {
					err := bCtx.StopContainer(ids[n])
					if err != nil {
						bCtx.Log().Warnf("got error on stopping container[%s]: %v", ids[n], err)
					}
				}

				return nil
			},
		})
	}
}

func genStartFunc(bCtx *boxship.Context, ids *[]string) func(desc boxship.ContainerDesc) {
	return func(desc boxship.ContainerDesc) {
		if desc.Disable {
			bCtx.Log().Warnf("[%s] is disabled", desc.Name)

			return
		}

		// check if Traefik is disabled to ignore starting it.
		if desc.Name == settings.Traefik && !bCtx.Settings().GetBool(settings.Traefik) {
			return
		}

		id, err := bCtx.RunContainer(desc.Name)
		if err != nil {
			bCtx.Log().Warnf("[%s] got error on running container: %v", desc.Name, err)
		} else {
			*ids = append(*ids, id)
		}
	}
}
