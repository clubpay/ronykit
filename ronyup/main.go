package main

import (
	"github.com/clubpay/ronykit/ronyup/cmd/setup"
	"github.com/clubpay/ronykit/ronyup/cmd/text"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use: "ronyup",
}

func main() {
	RootCmd.AddCommand(setup.Cmd, text.Cmd)
	err := RootCmd.Execute()
	if err != nil {
		panic(err)
	}
}
