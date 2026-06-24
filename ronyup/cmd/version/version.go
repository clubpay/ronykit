package version

import (
	"github.com/clubpay/ronykit/ronyup/internal"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Print the ronyup version",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.Println(internal.Version)

		return nil
	},
}
