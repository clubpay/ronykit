package mcp

import (
	"fmt"
	"net/http"

	"github.com/clubpay/ronykit/ronyup/cmd/mcp/knowledge"
	"github.com/clubpay/ronykit/ronyup/internal"
	"github.com/clubpay/ronykit/x/telemetry/logkit"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.Flags().Int("port", 0, "Port to run the MCP server over streamable HTTP. If 0, uses stdio")
}

//nolint:gochecknoglobals
var Cmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run ronyup as an MCP server over stdio or streamable HTTP",
	RunE: func(cmd *cobra.Command, _ []string) error {
		kb, err := knowledge.Load()
		if err != nil {
			return fmt.Errorf("load knowledge base: %w", err)
		}

		l := logkit.New()

		server := newServer(ServerConfig{
			name:         "RonyUP",
			version:      internal.Version,
			instructions: kb.ServerInstructions,
			kb:           kb,
			logger:       l.With("MCP").SLog(),
		})

		port, _ := cmd.Flags().GetInt("port")
		if port > 0 {
			handler := mcpsdk.NewStreamableHTTPHandler(
				func(_ *http.Request) *mcpsdk.Server {
					return server.srv
				},
				&mcpsdk.StreamableHTTPOptions{
					// Stateless enables protocol 2026-07-28 over HTTP and matches
					// this server's sessionless tool/resource model.
					Stateless: true,
				},
			)

			addr := fmt.Sprintf(":%d", port)
			cmd.PrintErrf("Starting MCP server on streamable HTTP %s\n", addr)

			return http.ListenAndServe(addr, handler)
		}

		return server.srv.Run(cmd.Context(), &mcpsdk.StdioTransport{})
	},
}
