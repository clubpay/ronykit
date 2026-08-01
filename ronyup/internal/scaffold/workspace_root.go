package scaffold

import (
	"fmt"
	"path/filepath"
)

// RunnerRelDir is the workspace-relative path of the shared runner module.
const RunnerRelDir = "pkg/runner"

const (
	runnerRelDir          = RunnerRelDir
	legacyCmdRunnerRelDir = "cmd/runner"
)

// ResolveGoWorkspace finds the directory that contains go.work.
// For backend workspaces that is the current directory; for fullstack workspaces
// it may be backend/ when invoked from the repository root.
func ResolveGoWorkspace(startDir string) (goRoot string, err error) {
	return resolveGoWorkspace(startDir)
}

func resolveGoWorkspace(startDir string) (goRoot string, err error) {
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	if fileExists(filepath.Join(abs, "go.work")) {
		return abs, nil
	}

	nested := filepath.Join(abs, backendDir, "go.work")
	if fileExists(nested) {
		return filepath.Join(abs, backendDir), nil
	}

	return "", fmt.Errorf(
		"run this command from the Go workspace root (directory with go.work) or from the repository root in a fullstack workspace (backend/go.work)",
	)
}

// RunnerDir returns goRoot/pkg/runner.
func RunnerDir(goRoot string) string { return runnerDir(goRoot) }

func runnerDir(goRoot string) string {
	return filepath.Join(goRoot, runnerRelDir)
}

// LegacyRunnerDir returns goRoot/internal/runner.
func LegacyRunnerDir(goRoot string) string { return legacyRunnerDir(goRoot) }

func legacyRunnerDir(goRoot string) string {
	return filepath.Join(goRoot, "internal", "runner")
}

// LegacyCmdRunnerDir returns goRoot/cmd/runner.
func LegacyCmdRunnerDir(goRoot string) string { return legacyCmdRunnerDir(goRoot) }

func legacyCmdRunnerDir(goRoot string) string {
	return filepath.Join(goRoot, legacyCmdRunnerRelDir)
}
