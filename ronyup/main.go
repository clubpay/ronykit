package main

import (
	"github.com/clubpay/ronykit/ronyup/cmd/mcp"
	"github.com/clubpay/ronykit/ronyup/cmd/setup"
	"github.com/clubpay/ronykit/ronyup/cmd/template"
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
	RootCmd.AddCommand(setup.Cmd, text.Cmd, template.Cmd, mcp.Cmd, version.Cmd)

	err := RootCmd.Execute()
	if err != nil {
		panic(err)
	}
}
