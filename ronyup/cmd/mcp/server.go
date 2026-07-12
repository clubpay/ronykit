package mcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/clubpay/ronykit/ronyup/cmd/mcp/knowledge"
	"github.com/clubpay/ronykit/ronyup/cmd/mcp/tools/scaffold"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type ServerConfig struct {
	name         string
	version      string
	instructions string
	executable   string
	skeletonFS   fs.FS
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
			InitializedHandler: func(ctx context.Context, request *mcpsdk.InitializedRequest) {
			},
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

// ---------------------------------------------------------------------------
// Shared validation helpers
// ---------------------------------------------------------------------------

var goIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func normalizeRelativePath(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", errors.New(errPathRequired)
	}

	cleanPath := path.Clean(strings.TrimPrefix(v, "/"))
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		return "", fmt.Errorf(errPathTraversal, v)
	}

	return cleanPath, nil
}

func normalizeFeatureName(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", errors.New(errFeatureNameRequired)
	}

	if !goIdentifierPattern.MatchString(v) {
		return "", fmt.Errorf(errFeatureNameInvalid, v)
	}

	return v, nil
}
