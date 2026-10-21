package setup

import (
	"encoding/json"
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

	assertClaudeHooks(t, filepath.Join(tmpDir, repoDir), true, false)

	for _, rel := range []string{
		"verify.sh", ".cursor/hooks/backend-verify.sh",
		".agents/skills/go-modern/SKILL.md",
	} {
		if _, err := os.Stat(filepath.Join(tmpDir, repoDir, rel)); err != nil {
			t.Fatalf("expected %s in backend-only workspace: %v", rel, err)
		}
	}

	assertClaudeSkillSymlink(t, filepath.Join(tmpDir, repoDir), "go-modern")
	assertClaudeSkillSymlink(t, filepath.Join(tmpDir, repoDir), "ronykit-framework")

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
		".agents/skills/go-modern/SKILL.md",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist at the repo root: %v", rel, err)
		}
	}

	assertClaudeSkillSymlink(t, root, "go-modern")
	assertClaudeSkillSymlink(t, root, "ronykit-framework")

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

	assertClaudeHooks(t, root, true, true)

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

	assertClaudeSkillSymlink(t, root, "frontend-design")
	assertClaudeSkillSymlink(t, root, "ronykit-framework")

	// The Go workspace and backend-only artifacts must NOT be scaffolded.
	for _, rel := range []string{
		"cmd", "feature", "pkg", "Makefile", "verify.sh", ".golangci.yml",
		"backend", ".cursor/hooks/backend-verify.sh",
		".agents/skills/go-modern",
		".cursor/skills/go-modern",
		".cursor/skills/frontend-design",
		".claude/skills/go-modern",
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

	assertClaudeHooks(t, root, false, true)

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

func TestSetupFeatureCommand_FromFullstackRepoRoot(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := "fs-feature-repo"
	repoModule := "github.com/example/fs-feature-repo"

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
	t.Cleanup(func() { opt = oldOpt })

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() unexpected error: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir(tmpDir) unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	opt.RepositoryRootDir = repoDir
	opt.RepositoryGoModule = repoModule
	opt.ApplicationName = "demo"
	opt.Kind = KindFullstack

	Cmd.SetOut(os.Stdout)
	Cmd.SetErr(os.Stderr)
	Cmd.SetArgs([]string{"workspace"})
	if err := Cmd.Execute(); err != nil {
		t.Fatalf("setup workspace: %v", err)
	}

	repoRoot := filepath.Join(tmpDir, repoDir)
	backend := filepath.Join(repoRoot, "backend")

	// Stub go exits 0 without creating files; seed the Go workspace markers
	// that resolveGoWorkspace / detectGoModule need.
	if err := os.WriteFile(filepath.Join(backend, "go.work"), []byte("go 1.25\n\nuse ./cmd/all-in-one\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.work): %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(backend, "cmd", "all-in-one", "go.mod"),
		[]byte("module "+repoModule+"/backend/cmd/all-in-one\n\ngo 1.25\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(go.mod): %v", err)
	}

	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir(repoRoot) unexpected error: %v", err)
	}

	opt.FeatureDir = "billing"
	opt.FeatureName = "billing"
	opt.Template = "service"
	opt.FeatureContainerFolder = "feature"
	opt.GroupByTemplate = false
	opt.Force = false

	Cmd.SetArgs([]string{"feature"})
	if err := Cmd.Execute(); err != nil {
		t.Fatalf("setup feature from fullstack repo root: %v", err)
	}

	wantFeature := filepath.Join(repoRoot, "backend", "feature", "billing")
	if _, err := os.Stat(wantFeature); err != nil {
		t.Fatalf("expected feature under backend/: %v", err)
	}

	wrongFeature := filepath.Join(repoRoot, "feature", "billing")
	if _, err := os.Stat(wrongFeature); !os.IsNotExist(err) {
		t.Fatalf("did not expect feature at repo root feature/billing (err=%v)", err)
	}

	featuresGo := filepath.Join(repoRoot, "backend", "cmd", "all-in-one", "features.go")
	content, err := os.ReadFile(featuresGo)
	if err != nil {
		t.Fatalf("ReadFile(features.go): %v", err)
	}
	if !strings.Contains(string(content), repoModule+"/backend/feature/billing") &&
		!strings.Contains(string(content), "feature/billing") {
		t.Fatalf("features.go should import the new feature, got:\n%s", content)
	}
}

func writeExecutable(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return err
	}

	return os.Chmod(path, 0o755)
}

// assertClaudeHooks checks .claude/settings.json is valid JSON and registers the backend
// PostToolUse (ronyup lint) and Stop (verify) hooks when backend is true, and the frontend
// Stop (frontend-verify) hook when frontend is true.
func assertClaudeHooks(t *testing.T, root string, backend, frontend bool) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile(.claude/settings.json) unexpected error: %v", err)
	}

	var settings struct {
		Permissions map[string]any `json:"permissions"`
		Hooks       map[string]any `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatalf(".claude/settings.json is not valid JSON: %v\n%s", err, raw)
	}

	if settings.Permissions == nil {
		t.Fatalf(".claude/settings.json lost its permissions block:\n%s", raw)
	}

	backendScripts := []string{".claude/hooks/backend-lint.sh", ".claude/hooks/backend-verify.sh"}
	frontendScripts := []string{".claude/hooks/frontend-verify.sh", ".cursor/hooks/frontend-verify.sh"}

	if backend {
		for _, event := range []string{"PostToolUse", "Stop"} {
			if settings.Hooks[event] == nil {
				t.Fatalf(".claude/settings.json should register a %s hook:\n%s", event, raw)
			}
		}

		for _, want := range []string{"backend-lint.sh", "backend-verify.sh", "Edit|MultiEdit|Write"} {
			if !strings.Contains(string(raw), want) {
				t.Fatalf(".claude/settings.json should mention %q:\n%s", want, raw)
			}
		}

		for _, rel := range backendScripts {
			if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
				t.Fatalf("expected %s: %v", rel, err)
			}
		}
	} else {
		if settings.Hooks["PostToolUse"] != nil {
			t.Fatalf("frontend-only .claude/settings.json should not register PostToolUse hook:\n%s", raw)
		}

		if strings.Contains(string(raw), "backend-verify.sh") {
			t.Fatalf("frontend-only .claude/settings.json should not mention backend-verify.sh:\n%s", raw)
		}

		for _, rel := range backendScripts {
			if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
				t.Fatalf("did not expect %s in frontend-only workspace (err=%v)", rel, err)
			}
		}
	}

	if frontend {
		if settings.Hooks["Stop"] == nil {
			t.Fatalf(".claude/settings.json should register a Stop hook:\n%s", raw)
		}

		if !strings.Contains(string(raw), "frontend-verify.sh") {
			t.Fatalf(".claude/settings.json should mention frontend-verify.sh:\n%s", raw)
		}

		for _, rel := range frontendScripts {
			if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
				t.Fatalf("expected %s: %v", rel, err)
			}
		}
	} else {
		if strings.Contains(string(raw), "frontend-verify.sh") {
			t.Fatalf("backend-only .claude/settings.json should not mention frontend-verify.sh:\n%s", raw)
		}

		for _, rel := range frontendScripts {
			if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
				t.Fatalf("did not expect %s in backend-only workspace (err=%v)", rel, err)
			}
		}
	}
}

func assertClaudeSkillSymlink(t *testing.T, root, id string) {
	t.Helper()

	link := filepath.Join(root, ".claude", "skills", id)

	got, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("claude skill %s should be a symlink: %v", id, err)
	}

	want := filepath.Join("..", "..", ".agents", "skills", id)
	if got != want {
		t.Fatalf("claude skill %s link = %q, want %q", id, got, want)
	}

	if _, err := os.Stat(filepath.Join(link, "SKILL.md")); err != nil {
		t.Fatalf("claude skill symlink %s does not resolve: %v", id, err)
	}

	cursor := filepath.Join(root, ".cursor", "skills", id)
	if _, err := os.Lstat(cursor); !os.IsNotExist(err) {
		t.Fatalf("did not expect .cursor/skills/%s (err=%v)", id, err)
	}
}
