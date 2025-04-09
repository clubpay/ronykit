package cmd

import (
	"os"

	"github.com/clubpay/ronykit/boxship/pkg/settings"

	"github.com/clubpay/ronykit/boxship/pkg/seeder"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var SeedCmd = &cobra.Command{
	Use:   "seed",
	Short: "is a helper utility to add seeder data",
	RunE: func(cmd *cobra.Command, args []string) error {
		yamlFile, err := cmd.Flags().GetString(settings.YamlFile)
		if err != nil {
			return err
		}

		yamlBytes, err := os.ReadFile(yamlFile)
		if err != nil {
			return err
		}

		seederConfig := seeder.Config{}
		err = yaml.Unmarshal(yamlBytes, &seederConfig)
		if err != nil {
			return err
		}

		return seederConfig.Seed()
	},
}

func init() {
	SeedCmd.Flags().String(settings.YamlFile, "", "the config yaml file")
}
