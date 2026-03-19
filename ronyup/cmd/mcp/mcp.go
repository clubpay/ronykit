package mcp

import (
	"io/fs"
	"os"

	"github.com/clubpay/ronykit/ronyup/internal"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var opt = struct {
	Name         string
	Version      string
	Instructions string
}{}

func init() {
	flagSet := Cmd.Flags()
	flagSet.StringVar(&opt.Name, "name", "ronyup", "MCP server name")
	flagSet.StringVar(&opt.Version, "version", "v0.1.0", "MCP server version")
	flagSet.StringVar(
		&opt.Instructions,
		"instructions",
		"RonyKIT scaffolding assistant. Follow layered service conventions: "+
			"keep API handlers thin, place business use-cases in internal/app, "+
			"use repo ports/adapters for persistence, and inspect templates "+
			"before generating or implementing modules.",
		"MCP server instructions advertised to clients",
	)
}

//nolint:gochecknoglobals
var Cmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run ronyup as an MCP server over stdio",
	RunE: func(cmd *cobra.Command, _ []string) error {
		exePath, err := os.Executable()
		if err != nil {
			return err
		}

		server := newServer(serverConfig{
			name:         opt.Name,
			version:      opt.Version,
			instructions: opt.Instructions,
			executable:   exePath,
			skeletonFS:   internal.Skeleton,
			cmdRunner:    defaultRunner{},
		})

		return server.Run(cmd.Context(), &mcpsdk.StdioTransport{})
	},
}

type serverConfig struct {
	name         string
	version      string
	instructions string
	executable   string
	skeletonFS   fs.FS
	cmdRunner    runner
}
