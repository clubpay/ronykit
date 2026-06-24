package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupWorkspaceCommand_DoesNotTemplateRenderMakefile(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := "sample-repo"
	repoModule := "github.com/example/sample-repo"

	stubBinDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(stubBinDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(stub bin) unexpected error: %v", err)
	}

	if err := writeExecutable(filepath.Join(stubBinDir, "go"), "#!/bin/sh\nexit 0\n"); err != nil {
		t.Fatalf("writeExecutable(go) unexpected error: %v", err)
	}

	if err := writeExecutable(filepath.Join(stubBinDir, "git"), "#!/bin/sh\nexit 0\n"); err != nil {
		t.Fatalf("writeExecutable(git) unexpected error: %v", err)
	}

	t.Setenv("PATH", stubBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	oldOpt := opt
	t.Cleanup(func() {
		opt = oldOpt
	})
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() unexpected error: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir(tmpDir) unexpected error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
	opt.RepositoryRootDir = repoDir
	opt.RepositoryGoModule = repoModule

	Cmd.SetOut(os.Stdout)
	Cmd.SetErr(os.Stderr)
	Cmd.SetArgs([]string{"workspace"})

	if err := Cmd.Execute(); err != nil {
		t.Fatalf("Cmd.Execute() unexpected error: %v", err)
	}

	makefilePath := filepath.Join(tmpDir, repoDir, "Makefile")
	makefileBytes, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Fatalf("ReadFile(Makefile) unexpected error: %v", err)
	}

	makefile := string(makefileBytes)
	if !strings.Contains(makefile, "{{.Dir}}") {
		t.Fatalf("Makefile template markers should stay untouched, got:\n%s", makefile)
	}
}

func TestSetupWorkspaceCommand_FullstackLayout(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := "fs-repo"
	repoModule := "github.com/example/fs-repo"

	stubBinDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(stubBinDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(stub bin) unexpected error: %v", err)
	}

	if err := writeExecutable(filepath.Join(stubBinDir, "go"), "#!/bin/sh\nexit 0\n"); err != nil {
		t.Fatalf("writeExecutable(go) unexpected error: %v", err)
	}

	if err := writeExecutable(filepath.Join(stubBinDir, "git"), "#!/bin/sh\nexit 0\n"); err != nil {
		t.Fatalf("writeExecutable(git) unexpected error: %v", err)
	}

	t.Setenv("PATH", stubBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	oldOpt := opt
	t.Cleanup(func() {
		opt = oldOpt
	})

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() unexpected error: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir(tmpDir) unexpected error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	opt.RepositoryRootDir = repoDir
	opt.RepositoryGoModule = repoModule
	opt.ApplicationName = "demo"
	opt.Kind = KindFullstack

	Cmd.SetOut(os.Stdout)
	Cmd.SetErr(os.Stderr)
	Cmd.SetArgs([]string{"workspace"})

	if err := Cmd.Execute(); err != nil {
		t.Fatalf("Cmd.Execute() unexpected error: %v", err)
	}

	root := filepath.Join(tmpDir, repoDir)

	// Go workspace lives under backend/.
	for _, rel := range []string{"backend/cmd/service", "backend/pkg/i18n", "backend/feature", "backend/Makefile", "backend/.golangci.yml"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist under backend: %v", rel, err)
		}
	}

	// Shared concerns stay at the repository root.
	for _, rel := range []string{"devops", "docs", "AGENTS.md", ".ai/mcp/mcp.json", "frontend/README.MD"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist at the repo root: %v", rel, err)
		}
	}

	// The Go workspace must not be duplicated at the repository root.
	if _, err := os.Stat(filepath.Join(root, "cmd")); !os.IsNotExist(err) {
		t.Fatalf("did not expect cmd/ at the repo root in fullstack mode (err=%v)", err)
	}

	// Backend Makefile must point at the root-level devops/ directory.
	makefile, err := os.ReadFile(filepath.Join(root, "backend", "Makefile"))
	if err != nil {
		t.Fatalf("ReadFile(backend Makefile) unexpected error: %v", err)
	}
	if !strings.Contains(string(makefile), "-f ../devops/docker-compose.yml") {
		t.Fatalf("backend Makefile should reference ../devops/docker-compose.yml, got:\n%s", makefile)
	}

	// Root AGENTS.md must render the fullstack guidance.
	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("ReadFile(AGENTS.md) unexpected error: %v", err)
	}
	if !strings.Contains(string(agents), "Fullstack project") {
		t.Fatalf("AGENTS.md should render fullstack guidance, got:\n%s", agents)
	}
}

func writeExecutable(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return err
	}

	return os.Chmod(path, 0o755)
}
