package mcp

import (
	"log/slog"

	"github.com/clubpay/ronykit/ronyup/cmd/mcp/knowledge"
	"github.com/clubpay/ronykit/ronyup/cmd/mcp/tools/scaffold"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type ServerConfig struct {
	name         string
	version      string
	instructions string
	kb           *knowledge.Base
	logger       *slog.Logger
}

type Server struct {
	cfg ServerConfig
	srv *mcpsdk.Server
}

func newServer(cfg ServerConfig) *Server {
	srv := mcpsdk.NewServer(
		&mcpsdk.Implementation{
			Name:    cfg.name,
			Version: cfg.version,
			Title:   serverTitle,
		},
		&mcpsdk.ServerOptions{
			Instructions:      cfg.instructions,
			CompletionHandler: completionHandler(cfg.kb),
			Logger:            cfg.logger,
			// Empty Capabilities disables the historical default {"logging":{}}
			// advertisement (logging is deprecated as of protocol 2026-07-28).
			// tools/prompts/resources capabilities are still inferred from
			// registered features.
			Capabilities: &mcpsdk.ServerCapabilities{},
		},
	)

	registerResources(srv, cfg)
	registerPrompts(srv, cfg)

	scaffold.Register(srv)

	return &Server{
		cfg: cfg,
		srv: srv,
	}
}
