package setup

import (
	"fmt"
	"path/filepath"
)

const runnerRelDir = "pkg/runner"

const legacyCmdRunnerRelDir = "cmd/runner"

// resolveGoWorkspace finds the directory that contains go.work.
// For backend workspaces that is the current directory; for fullstack workspaces
// it may be backend/ when invoked from the repository root.
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

func runnerDir(goRoot string) string {
	return filepath.Join(goRoot, runnerRelDir)
}

func legacyRunnerDir(goRoot string) string {
	return filepath.Join(goRoot, "internal", "runner")
}

func legacyCmdRunnerDir(goRoot string) string {
	return filepath.Join(goRoot, legacyCmdRunnerRelDir)
}
