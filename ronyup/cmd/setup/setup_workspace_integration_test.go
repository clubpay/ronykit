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

func writeExecutable(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return err
	}

	return os.Chmod(path, 0o755)
}
