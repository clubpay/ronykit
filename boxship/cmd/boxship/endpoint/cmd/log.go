package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/clubpay/ronykit/boxship"
	"github.com/clubpay/ronykit/boxship/pkg/log"
	"github.com/clubpay/ronykit/boxship/pkg/settings"
	"github.com/clubpay/ronykit/boxship/pkg/settings/viper"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

var LogCmd = &cobra.Command{
	Use:          "log",
	Short:        "print log of the containers",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			cmd.Println("you must specify a container")

			return nil
		}
		var set settings.Settings
		app := fx.New(
			fx.NopLogger,
			fx.Supply(
				fx.Annotate(log.New(-1), fx.As(new(log.Logger))),
				cmd,
			),
			fx.Provide(viper.New, boxship.NewContext),
			fx.StartTimeout(time.Hour),
			fx.Decorate(settings.PrepareDefaults),
			fx.Populate(&set),
		)
		err := app.Start(context.TODO())
		if err != nil {
			return err
		}

		filePath := fmt.Sprintf("%s.log", settings.GetLogsDir(set, args[0]))
		if b, _ := cmd.Flags().GetBool("build"); b {
			filePath = fmt.Sprintf("%s-build.log", settings.GetLogsDir(set, args[0]))
		}

		data, err := os.ReadFile(filePath)
		if err == nil {
			cmd.Println(string(data))
		}

		return app.Stop(context.TODO())
	},
}

func init() {
	LogCmd.Flags().Bool("build", false, "show build logs")
	LogCmd.Flags().Bool("tail", false, "tail logs")
}
