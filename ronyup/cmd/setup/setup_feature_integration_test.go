package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupFeatureCommand_CreatesBundledConfig(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := "feature-repo"
	repoModule := "github.com/example/feature-repo"

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
		t.Fatalf("setup workspace: %v", err)
	}

	workspaceRoot := filepath.Join(tmpDir, repoDir)
	if err := os.Chdir(workspaceRoot); err != nil {
		t.Fatalf("Chdir(workspaceRoot) unexpected error: %v", err)
	}

	if err := os.WriteFile(filepath.Join(workspaceRoot, "go.work"), []byte("go 1.25\n\nuse ./cmd/service\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.work) unexpected error: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(workspaceRoot, "cmd", "service", "go.mod"),
		[]byte("module "+repoModule+"/cmd/service\n\ngo 1.25\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(cmd/service/go.mod) unexpected error: %v", err)
	}

	opt.RepositoryRootDir = workspaceRoot
	opt.FeatureDir = "auth"
	opt.FeatureName = "auth"
	opt.Template = "service"
	opt.FeatureContainerFolder = "feature"
	opt.GroupByTemplate = false

	Cmd.SetArgs([]string{
		"feature",
		"-m", repoModule,
		"-p", "auth",
		"-n", "auth",
		"-t", "service",
	})
	if err := Cmd.Execute(); err != nil {
		t.Fatalf("setup feature: %v", err)
	}

	configPath := filepath.Join(workspaceRoot, "config", "service", "auth.local.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(bundled config) unexpected error: %v", err)
	}

	config := string(data)
	if !strings.Contains(config, "auth-db") {
		t.Fatalf("bundled config should render feature db name, got:\n%s", config)
	}

	standalonePath := filepath.Join(workspaceRoot, "feature", "auth", "internal", "settings", "config.local.yaml")
	if _, err := os.Stat(standalonePath); err != nil {
		t.Fatalf("expected standalone config at %s: %v", standalonePath, err)
	}
}
