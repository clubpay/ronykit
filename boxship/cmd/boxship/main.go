package main

import (
	"github.com/clubpay/ronykit/boxship/cmd/boxship/endpoint/cmd"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load(".env.local")
	}

	cmd.RootCmd.AddCommand(
		cmd.CertCmd, cmd.SeedCmd,
		cmd.InitCmd, cmd.BuildCmd, cmd.RunCmd, cmd.BuildNetworkCmd, cmd.SetupDNSCmd,
		cmd.ListCmd, cmd.UpdateTemplateCmd,
		cmd.VersionCmd, cmd.LogCmd, cmd.UpdateCmd,
	)

	err := cmd.RootCmd.Execute()
	if err != nil {
		panic(err)
	}
}
