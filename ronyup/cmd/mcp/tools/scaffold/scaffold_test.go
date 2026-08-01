package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	appscaffold "github.com/clubpay/ronykit/ronyup/internal/scaffold"
)

func TestSetupWorkspace_InProcessCreatesAtPath(t *testing.T) {
	tmp := t.TempDir()
	dest := filepath.Join(tmp, "ws")

	stubBin := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(stubBin, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"go", "git"} {
		path := filepath.Join(stubBin, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", stubBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := appscaffold.SetupWorkspace(context.Background(), appscaffold.WorkspaceRequest{
		Path:   dest,
		Module: "github.com/example/ws",
		Kind:   appscaffold.KindBackend,
		Skills: []string{appscaffold.SkillTokenNone},
	}, appscaffold.DiscardLogger{})
	if err != nil {
		t.Fatalf("SetupWorkspace: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, "AGENTS.md")); err != nil {
		t.Fatalf("expected workspace at %s, not a nested my-repo: %v", dest, err)
	}
	if _, err := os.Stat(filepath.Join(dest, "my-repo")); !os.IsNotExist(err) {
		t.Fatalf("did not expect nested my-repo under destination")
	}
}
