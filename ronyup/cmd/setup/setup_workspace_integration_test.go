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

	// Backend-only workspaces get the backend verify stop hook (no frontend hook).
	hooks, err := os.ReadFile(filepath.Join(tmpDir, repoDir, ".cursor", "hooks.json"))
	if err != nil {
		t.Fatalf("ReadFile(.cursor/hooks.json) unexpected error: %v", err)
	}
	if strings.Contains(string(hooks), "frontend-verify.sh") {
		t.Fatalf("backend-only hooks.json should not register the frontend stop hook, got:\n%s", hooks)
	}
	if !strings.Contains(string(hooks), "backend-verify.sh") {
		t.Fatalf("backend-only hooks.json should register the backend stop hook, got:\n%s", hooks)
	}

	for _, rel := range []string{"verify.sh", ".cursor/hooks/backend-verify.sh"} {
		if _, err := os.Stat(filepath.Join(tmpDir, repoDir, rel)); err != nil {
			t.Fatalf("expected %s in backend-only workspace: %v", rel, err)
		}
	}

	for _, rel := range []string{
		"devops/devbox/Makefile", "devops/devbox/Vagrantfile", "devops/devbox/config.yaml",
		"bundles.yaml", "pkg/runner/runner.go", "cmd/all-in-one/main.go",
	} {
		if _, err := os.Stat(filepath.Join(tmpDir, repoDir, rel)); err != nil {
			t.Fatalf("expected %s in backend-only workspace: %v", rel, err)
		}
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
	for _, rel := range []string{"backend/cmd/all-in-one", "backend/pkg/i18n", "backend/pkg/runner", "backend/bundles.yaml", "backend/feature", "backend/Makefile", "backend/.golangci.yml"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist under backend: %v", rel, err)
		}
	}

	// Shared concerns stay at the repository root, and the frontend verify
	// gate + enforcing stop hook are seeded.
	for _, rel := range []string{
		"devops", "docs", "AGENTS.md", ".ai/mcp/mcp.json",
		"devops/devbox/Makefile", "devops/devbox/Vagrantfile", "devops/devbox/config.yaml",
		"frontend/README.MD", "frontend/verify.sh", "frontend/Makefile",
		"backend/verify.sh", "backend/Makefile",
		".cursor/hooks.json", ".cursor/hooks/frontend-verify.sh", ".cursor/hooks/backend-verify.sh",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist at the repo root: %v", rel, err)
		}
	}

	// The fullstack stop hook must be wired to the frontend verify script.
	hooks, err := os.ReadFile(filepath.Join(root, ".cursor", "hooks.json"))
	if err != nil {
		t.Fatalf("ReadFile(.cursor/hooks.json) unexpected error: %v", err)
	}
	if !strings.Contains(string(hooks), "frontend-verify.sh") {
		t.Fatalf("fullstack hooks.json should register the frontend-verify stop hook, got:\n%s", hooks)
	}
	if !strings.Contains(string(hooks), "backend-verify.sh") {
		t.Fatalf("fullstack hooks.json should register the backend-verify stop hook, got:\n%s", hooks)
	}

	// The Go workspace must not be duplicated at the repository root.
	if _, err := os.Stat(filepath.Join(root, "cmd")); !os.IsNotExist(err) {
		t.Fatalf("did not expect cmd/ at the repo root in fullstack mode (err=%v)", err)
	}

	// Backend Makefile must expose devbox helpers pointing at root devops/devbox.
	makefile, err := os.ReadFile(filepath.Join(root, "backend", "Makefile"))
	if err != nil {
		t.Fatalf("ReadFile(backend Makefile) unexpected error: %v", err)
	}
	if !strings.Contains(string(makefile), "../devops/devbox") {
		t.Fatalf("backend Makefile should reference ../devops/devbox, got:\n%s", makefile)
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

func TestSetupWorkspaceCommand_FrontendOnlyLayout(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := "fe-repo"
	repoModule := "github.com/example/fe-repo"

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
	opt.Kind = KindFrontend

	Cmd.SetOut(os.Stdout)
	Cmd.SetErr(os.Stderr)
	Cmd.SetArgs([]string{"workspace"})

	if err := Cmd.Execute(); err != nil {
		t.Fatalf("Cmd.Execute() unexpected error: %v", err)
	}

	root := filepath.Join(tmpDir, repoDir)

	// Frontend app, shared AI config, docs/, and the frontend stop hook exist.
	for _, rel := range []string{
		"frontend/README.MD", "frontend/verify.sh", "frontend/Makefile",
		"docs", "AGENTS.md", ".ai/mcp/mcp.json",
		".cursor/hooks.json", ".cursor/hooks/frontend-verify.sh",
		".agents/skills/frontend-design",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s in frontend-only workspace: %v", rel, err)
		}
	}

	// The Go workspace and backend-only artifacts must NOT be scaffolded.
	for _, rel := range []string{
		"cmd", "feature", "pkg", "Makefile", "verify.sh", ".golangci.yml",
		"backend", ".cursor/hooks/backend-verify.sh", ".agents/skills/go-modern",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			t.Fatalf("did not expect %s in frontend-only workspace (err=%v)", rel, err)
		}
	}

	// Stop hook wires only the frontend verify gate.
	hooks, err := os.ReadFile(filepath.Join(root, ".cursor", "hooks.json"))
	if err != nil {
		t.Fatalf("ReadFile(.cursor/hooks.json) unexpected error: %v", err)
	}
	if !strings.Contains(string(hooks), "frontend-verify.sh") {
		t.Fatalf("frontend-only hooks.json should register the frontend stop hook, got:\n%s", hooks)
	}
	if strings.Contains(string(hooks), "backend-verify.sh") {
		t.Fatalf("frontend-only hooks.json should not register the backend stop hook, got:\n%s", hooks)
	}

	// AGENTS.md renders frontend-only guidance and drops backend sections.
	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("ReadFile(AGENTS.md) unexpected error: %v", err)
	}
	if !strings.Contains(string(agents), "Frontend-only project") {
		t.Fatalf("AGENTS.md should render frontend-only guidance, got:\n%s", agents)
	}
	if strings.Contains(string(agents), "## Package Selection (mandatory)") {
		t.Fatalf("frontend-only AGENTS.md should not include the backend package-selection section, got:\n%s", agents)
	}
}

func writeExecutable(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return err
	}

	return os.Chmod(path, 0o755)
}
