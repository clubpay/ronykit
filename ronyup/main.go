package main

import (
	"github.com/clubpay/ronykit/ronyup/cmd/setup"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use: "ronyup",
}

func main() {
	RootCmd.AddCommand(setup.Cmd)
	err := RootCmd.Execute()
	if err != nil {
		panic(err)
	}
}
