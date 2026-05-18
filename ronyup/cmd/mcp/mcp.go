package mcp

import (
	"fmt"
	"net/http"
	"os"

	"github.com/clubpay/ronykit/ronyup/cmd/mcp/knowledge"
	"github.com/clubpay/ronykit/ronyup/internal"
	"github.com/clubpay/ronykit/x/telemetry/logkit"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.Flags().Int("port", 0, "Port to run the MCP server over HTTP/SSE. If 0, uses stdio")
}

//nolint:gochecknoglobals
var Cmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run ronyup as an MCP server over stdio or HTTP/SSE",
	RunE: func(cmd *cobra.Command, _ []string) error {
		kb, err := knowledge.Load()
		if err != nil {
			return fmt.Errorf("load knowledge base: %w", err)
		}

		exePath, err := os.Executable()
		if err != nil {
			return err
		}

		l := logkit.New()

		server := newServer(ServerConfig{
			name:         "RonyUP",
			version:      "v0.0.1",
			instructions: kb.ServerInstructions,
			executable:   exePath,
			skeletonFS:   internal.Skeleton,
			cmdRunner:    defaultRunner{},
			kb:           kb,
			logger:       l.With("MCP").SLog(),
		})

		port, _ := cmd.Flags().GetInt("port")
		if port > 0 {
			handler := mcpsdk.NewSSEHandler(
				func(request *http.Request) *mcpsdk.Server {
					return server.srv
				},
				nil,
			)

			addr := fmt.Sprintf(":%d", port)
			cmd.PrintErrf("Starting MCP server on HTTP %s\n", addr)

			return http.ListenAndServe(addr, handler)
		}

		return server.srv.Run(cmd.Context(), &mcpsdk.StdioTransport{})
	},
}
