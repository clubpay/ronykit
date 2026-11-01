package main

import (
	"os"

	"github.com/clubpay/ronykit/ronyup/cmd/lint"
	"github.com/clubpay/ronykit/ronyup/cmd/mcp"
	"github.com/clubpay/ronykit/ronyup/cmd/setup"
	"github.com/clubpay/ronykit/ronyup/cmd/text"
	"github.com/clubpay/ronykit/ronyup/cmd/version"
	"github.com/clubpay/ronykit/ronyup/internal"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:     "ronyup",
	Version: internal.Version,
}

func main() {
	RootCmd.AddCommand(setup.Cmd, text.Cmd, mcp.Cmd, lint.Cmd, version.Cmd)

	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
