package mcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolRegistrar registers a single MCP tool on the server.
// Adding a new tool requires writing a registrar function and
// appending it to allTools.
type toolRegistrar func(srv *mcpsdk.Server, cfg serverConfig)

func allTools() []toolRegistrar {
	return []toolRegistrar{
		registerListTemplates,
		registerReadTemplate,
		registerCreateWorkspace,
		registerCreateFeature,
		registerPlanService,
		registerImplementService,
	}
}

func newServer(cfg serverConfig) *mcpsdk.Server {
	srv := mcpsdk.NewServer(
		&mcpsdk.Implementation{
			Name:    cfg.name,
			Version: cfg.version,
			Title:   serverTitle,
		},
		&mcpsdk.ServerOptions{
			Instructions: cfg.instructions,
		},
	)

	for _, register := range allTools() {
		register(srv, cfg)
	}

	registerResources(srv, cfg)
	registerPrompts(srv, cfg)

	return srv
}

// ---------------------------------------------------------------------------
// Command runner abstraction
// ---------------------------------------------------------------------------

type runner interface {
	Run(ctx context.Context, cwd, name string, args ...string) (stdout, stderr string, err error)
}

type defaultRunner struct{}

func (defaultRunner) Run(ctx context.Context, cwd, name string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = cwd

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

// ---------------------------------------------------------------------------
// Workspace inspection (shared by plan_service and implement_service)
// ---------------------------------------------------------------------------

type workspaceInspection struct {
	AbsDir                  string
	ExistingServiceFeatures []string
}

func inspectWorkspace(workspaceDir string) (workspaceInspection, error) {
	absDir, err := filepath.Abs(workspaceDir)
	if err != nil {
		return workspaceInspection{}, err
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return workspaceInspection{}, err
	}

	if !info.IsDir() {
		return workspaceInspection{}, fmt.Errorf(errWorkspaceNotDir, workspaceDir)
	}

	_, err = os.Stat(filepath.Join(absDir, "go.work"))
	if err != nil {
		return workspaceInspection{}, fmt.Errorf(errWorkspaceNoGoWork, err)
	}

	featuresRoot := filepath.Join(absDir, "feature", "service")

	entries, err := os.ReadDir(featuresRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return workspaceInspection{AbsDir: absDir}, nil
		}

		return workspaceInspection{}, err
	}

	existing := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		existing = append(existing, entry.Name())
	}

	sort.Strings(existing)

	return workspaceInspection{
		AbsDir:                  absDir,
		ExistingServiceFeatures: existing,
	}, nil
}
