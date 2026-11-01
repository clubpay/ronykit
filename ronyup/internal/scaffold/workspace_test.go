package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInitGoWorkspaceDisablesOuterWorkspaceForInit(t *testing.T) {
	root := t.TempDir()
	for _, pkg := range []string{"pkg/i18n", "pkg/runner", "cmd/all-in-one"} {
		if err := os.MkdirAll(filepath.Join(root, pkg), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	bin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
if [ "$1" = "work" ] && [ "$2" = "init" ] && [ "${GOWORK:-}" != "off" ]; then
  exit 9
fi
exit 0
`
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := initGoWorkspace(context.Background(), workspaceCopyInput{
		repoRoot: root,
		module:   "github.com/example/ws",
		kind:     KindBackend,
	})
	if err != nil {
		t.Fatalf("initGoWorkspace: %v", err)
	}
}
