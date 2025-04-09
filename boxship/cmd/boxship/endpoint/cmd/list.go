package cmd

import (
	"context"
	"time"

	"github.com/clubpay/ronykit/boxship"
	"github.com/clubpay/ronykit/boxship/pkg/log"
	"github.com/clubpay/ronykit/boxship/pkg/settings"
	"github.com/clubpay/ronykit/boxship/pkg/settings/viper"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

var ListCmd = &cobra.Command{
	Use:          "list",
	Short:        "list the containers in setup",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := fx.New(
			fx.NopLogger,
			fx.Supply(
				fx.Annotate(log.New(-1), fx.As(new(log.Logger))),
				cmd,
			),
			fx.Provide(viper.New, boxship.NewContext),
			fx.StartTimeout(time.Hour),
			fx.Decorate(settings.PrepareDefaults),
			fx.Invoke(list),
		)
		err := app.Start(context.TODO())
		if err != nil {
			return err
		}

		return app.Stop(context.TODO())
	},
}

func list(lc fx.Lifecycle, bCtx *boxship.Context) {
	lc.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
				t := table.New("Index", "Name", "Disabled", "Image").
					WithHeaderFormatter(headerFmt)

				bCtx.ForEachContainer(
					func(desc boxship.ContainerDesc) {
						t.AddRow(desc.Index, desc.Name, desc.Disable, desc.GetImage())
					},
				)

				t.Print()

				return nil
			},
		},
	)
}
