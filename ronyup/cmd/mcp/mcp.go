package mcp

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/clubpay/ronykit/ronyup/internal"
	"github.com/clubpay/ronykit/ronyup/knowledge"
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
		"",
		"MCP server instructions advertised to clients (defaults to embedded knowledge)",
	)
}

//nolint:gochecknoglobals
var Cmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run ronyup as an MCP server over stdio",
	RunE: func(cmd *cobra.Command, _ []string) error {
		kb, err := knowledge.Load()
		if err != nil {
			return fmt.Errorf("load knowledge base: %w", err)
		}

		exePath, err := os.Executable()
		if err != nil {
			return err
		}

		instructions := opt.Instructions
		if instructions == "" {
			instructions = kb.ServerInstructions
		}

		server := newServer(serverConfig{
			name:         opt.Name,
			version:      opt.Version,
			instructions: instructions,
			executable:   exePath,
			skeletonFS:   internal.Skeleton,
			cmdRunner:    defaultRunner{},
			kb:           kb,
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
	kb           *knowledge.Base
}
