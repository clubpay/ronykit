package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/clubpay/ronykit/boxship"
	"github.com/clubpay/ronykit/boxship/pkg/log"
	"github.com/clubpay/ronykit/boxship/pkg/settings"
	"github.com/clubpay/ronykit/boxship/pkg/settings/viper"
	"github.com/spf13/cobra"
	"github.com/txn2/txeh"
	"go.uber.org/fx"
)

func init() {
	SetupDNSCmd.Flags().String(settings.Setup, "./setup", "")
}

var SetupDNSCmd = &cobra.Command{
	Use:          "setup-dns",
	Short:        "creates the dns records locally using /etc/hosts or the provider",
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
			fx.Invoke(setupDNS()),
		)
		err := app.Start(context.TODO())
		if err != nil {
			return err
		}

		return app.Stop(context.TODO())
	},
}

func setupDNS() func(lc fx.Lifecycle, bCtx *boxship.Context) {
	return func(lc fx.Lifecycle, bCtx *boxship.Context) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				hosts, err := txeh.NewHostsDefault()
				if err != nil {
					return err
				}

				var domains []string
				bCtx.ForEachContainer(
					func(desc boxship.ContainerDesc) {
						if desc.Disable || desc.HTTPRoute == nil {
							return
						}

						domains = append(domains, fmt.Sprintf("%s.%s", desc.HTTPRoute.SubDomain, bCtx.Domain()))
					},
				)

				if len(domains) == 0 {
					return nil
				}

				bCtx.Log().Infof("adding the following domains: %v", domains)
				hosts.AddHosts("127.0.0.1", domains)
				err = hosts.Save()
				if err != nil {
					return err
				}

				err = hosts.Reload()
				if err != nil {
					return err
				}
				bCtx.Log().Infof("hosts file updated: \r\n %s", hosts.RenderHostsFile())

				return nil
			},
		})
	}
}
