package cmd

import (
	"github.com/clubpay/ronykit/boxship"
	"github.com/spf13/cobra"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "current version",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println(boxship.Version.String())

		return nil
	},
}

func init() {}
