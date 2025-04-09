package viper

import (
	"strings"

	"github.com/joho/godotenv"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/clubpay/ronykit/boxship/pkg/settings"
	"github.com/spf13/viper"
)

var _ settings.Settings = Settings{}

type Config struct {
	fx.In

	Command *cobra.Command `optional:"true"`
}

type Settings struct {
	*viper.Viper
}

func New(cfg Config) settings.Settings {
	_ = godotenv.Load()

	v := viper.New()

	if cfg.Command != nil {
		_ = v.BindPFlags(cfg.Command.Flags())
	}

	// Read Parameters from environment
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	return &Settings{
		Viper: v,
	}
}
