package main

import (
	"ronyup/cmd/setup"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use: "ronyup",
}

func main() {
	RootCmd.AddCommand(setup.Cmd)
	_ = RootCmd.Execute()
}
