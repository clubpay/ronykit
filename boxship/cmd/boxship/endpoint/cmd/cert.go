package cmd

import (
	"github.com/clubpay/ronykit/boxship/pkg/settings"
	"github.com/clubpay/ronykit/boxship/pkg/utils"
	"github.com/spf13/cobra"
)

var CertCmd = &cobra.Command{
	Use:   "gen-root-ca",
	Short: "is helper utility to generate root certificate",
	RunE: func(cmd *cobra.Command, args []string) error {
		output, err := cmd.Flags().GetString(settings.OutputDir)
		if err != nil {
			return err
		}

		rootCertTmpl := utils.RootCATemplate("US", "BoxShip", "Ronak Software Group Inc.")
		utils.GenerateRootCertificate(rootCertTmpl, output)

		return nil
	},
}

func init() {
	CertCmd.Flags().String(settings.CACertFile, "./setup/rootca/ca.crt", "")
	CertCmd.Flags().String(settings.CAKeyFile, "./setup/rootca/ca.key", "")
	CertCmd.Flags().String(
		settings.OutputDir, "./setup/rootca/",
		"the folder that root certificate is going to be generated",
	)
}
