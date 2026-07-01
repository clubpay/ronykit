package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSetupSyncCommand_AddsMissingDevbox(t *testing.T) {
	root := scaffoldLegacyBackendWorkspace(t)

	oldOpt := opt
	oldSync := syncOpt
	t.Cleanup(func() {
		opt = oldOpt
		syncOpt = oldSync
	})

	chdir(t, root)

	syncOpt = struct {
		RepoDir    string
		Kind       string
		Only       []string
		Overwrite  bool
		SkillsMode []string
	}{
		RepoDir: ".",
		Only:    []string{syncSectionDevops},
	}

	if err := runSync(newSilentCommand(t)); err != nil {
		t.Fatalf("runSync(devops): %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "devops", "devbox", "Makefile")); err != nil {
		t.Fatalf("expected devbox Makefile after sync: %v", err)
	}

	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("ReadFile(AGENTS.md): %v", err)
	}

	if string(agents) != "# legacy\n" {
		t.Fatalf("sync without overwrite should keep AGENTS.md, got:\n%s", agents)
	}
}

func TestSetupSyncCommand_OverwriteAgents(t *testing.T) {
	root := scaffoldLegacyBackendWorkspace(t)

	oldOpt := opt
	oldSync := syncOpt
	t.Cleanup(func() {
		opt = oldOpt
		syncOpt = oldSync
	})

	chdir(t, root)

	opt.ApplicationName = "legacy"
	opt.RepositoryGoModule = "github.com/example/legacy-repo"

	syncOpt = struct {
		RepoDir    string
		Kind       string
		Only       []string
		Overwrite  bool
		SkillsMode []string
	}{
		RepoDir:   ".",
		Only:      []string{syncSectionAgents},
		Overwrite: true,
	}

	if err := runSync(newSilentCommand(t)); err != nil {
		t.Fatalf("runSync(agents): %v", err)
	}

	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("ReadFile(AGENTS.md): %v", err)
	}

	if !strings.Contains(string(agents), "RonyKIT") {
		t.Fatalf("expected rendered AGENTS.md after overwrite, got:\n%s", agents)
	}
}

func scaffoldLegacyBackendWorkspace(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	for _, dir := range []string{"cmd/service", "feature"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", dir, err)
		}
	}

	files := map[string]string{
		"go.work":            "go 1.25\n\nuse ./cmd/service\n",
		"cmd/service/go.mod": "module github.com/example/legacy-repo/cmd/service\n\ngo 1.25\n",
		"AGENTS.md":          "# legacy\n",
	}

	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", rel, err)
		}
	}

	return root
}

func chdir(t *testing.T, dir string) {
	t.Helper()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	t.Cleanup(func() { _ = os.Chdir(oldWD) })
}

func newSilentCommand(t *testing.T) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{}
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	return cmd
}
