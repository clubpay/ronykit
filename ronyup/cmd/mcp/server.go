package mcp

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"

	"github.com/clubpay/ronykit/ronyup/cmd/mcp/knowledge"
	"github.com/clubpay/ronykit/ronyup/cmd/mcp/tools/scaffold"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type ServerConfig struct {
	name         string
	version      string
	instructions string
	executable   string
	cmdRunner    runner
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

	scaffold.Register(srv, cfg.cmdRunner, cfg.executable)

	return &Server{
		cfg: cfg,
		srv: srv,
	}
}

// ---------------------------------------------------------------------------
// Command runner abstraction
// ---------------------------------------------------------------------------

type runner interface {
	Run(ctx context.Context, cwd, name string, args ...string) (stdout, stderr string, err error)
}

type defaultRunner struct {
	envs []string
}

func (r defaultRunner) Run(ctx context.Context, cwd, name string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	cmd.Dir = cwd
	if len(r.envs) > 0 {
		cmd.Env = append(os.Environ(), r.envs...)
	}

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}
