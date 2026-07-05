package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectBundleLayout_Legacy(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	serviceDir := filepath.Join(root, "cmd", "service")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	legacyMain := `package main

import "github.com/spf13/cobra"

func genServerProvider(host string, port int) {}
var rootCmd = &cobra.Command{}
func main() {}
`
	if err := os.WriteFile(filepath.Join(serviceDir, "main.go"), []byte(legacyMain), 0o644); err != nil {
		t.Fatalf("WriteFile main.go: %v", err)
	}

	for _, name := range []string{"middleware.go", "healthz.go"} {
		if err := os.WriteFile(filepath.Join(serviceDir, name), []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	status := detectBundleLayout(root)

	if status.IsCurrent() {
		t.Fatal("expected legacy workspace to need migration")
	}

	if !status.LegacyMain || !status.LegacyMiddleware || !status.LegacyHealthz {
		t.Fatalf("unexpected status: %+v", status)
	}
}

func TestDetectBundleLayout_Current(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	runnerDir := filepath.Join(root, "internal", "runner")
	serviceDir := filepath.Join(root, "cmd", "service")

	for _, dir := range []string{runnerDir, serviceDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
	}

	currentMain := `package main

import "github.com/example/app/internal/runner"

func main() {
	runner.Execute(runner.Config{})
}
`
	files := map[string]string{
		filepath.Join(runnerDir, "runner.go"): "package runner\n",
		filepath.Join(runnerDir, "go.mod"):    "module github.com/example/app/internal/runner\n\ngo 1.25\n",
		filepath.Join(serviceDir, "main.go"):  currentMain,
		bundlesManifestPath(root):             "bundles:\n  service:\n    services: [\"*\"]\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}

	status := detectBundleLayout(root)

	if !status.IsCurrent() {
		t.Fatalf("expected current workspace, got: %+v", status)
	}
}

func TestBuildMigratePlan_Legacy(t *testing.T) {
	t.Parallel()

	status := bundleLayoutStatus{
		LegacyMain:       true,
		LegacyMiddleware: true,
		LegacyHealthz:    true,
	}

	plan := buildMigratePlan(status)
	joined := strings.Join(plan, "\n")

	for _, want := range []string{
		"copy internal/runner/",
		"rewrite cmd/service/main.go",
		"remove cmd/service/middleware.go",
		"remove cmd/service/healthz.go",
		"create bundles.yaml",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("plan missing %q:\n%s", want, joined)
		}
	}
}

func TestRunMigrateBundles_DryRunLegacy(t *testing.T) {
	root := scaffoldLegacyBundleWorkspace(t)

	oldOpt := opt
	t.Cleanup(func() { opt = oldOpt })

	chdir(t, root)
	opt.RepositoryGoModule = "github.com/example/legacy-repo"

	cmd := newSilentCommand(t)
	migrateOpt.DryRun = true
	t.Cleanup(func() { migrateOpt.DryRun = false })

	if err := runMigrateBundles(cmd); err != nil {
		t.Fatalf("runMigrateBundles(dry-run): %v", err)
	}

	if fileExists(filepath.Join(root, "internal", "runner", "runner.go")) {
		t.Fatal("dry-run should not create internal/runner")
	}
}

func TestRunMigrateBundles_UpgradesLegacyWorkspace(t *testing.T) {
	root := scaffoldLegacyBundleWorkspace(t)

	stubBinDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(stubBinDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(stub bin): %v", err)
	}

	if err := writeExecutable(filepath.Join(stubBinDir, "go"), "#!/bin/sh\nexit 0\n"); err != nil {
		t.Fatalf("writeExecutable(go): %v", err)
	}

	t.Setenv("PATH", stubBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	oldOpt := opt
	t.Cleanup(func() { opt = oldOpt })

	chdir(t, root)
	opt.RepositoryGoModule = "github.com/example/legacy-repo"

	cmd := newSilentCommand(t)
	if err := runMigrateBundles(cmd); err != nil {
		t.Fatalf("runMigrateBundles(): %v", err)
	}

	for _, rel := range []string{
		"internal/runner/runner.go",
		"bundles.yaml",
		"cmd/service/main.go",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s after migration: %v", rel, err)
		}
	}

	for _, rel := range []string{"cmd/service/middleware.go", "cmd/service/healthz.go"} {
		if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			t.Fatalf("expected %s removed after migration: %v", rel, err)
		}
	}

	main, err := os.ReadFile(filepath.Join(root, "cmd/service/main.go"))
	if err != nil {
		t.Fatalf("ReadFile main.go: %v", err)
	}

	if !strings.Contains(string(main), "internal/runner") {
		t.Fatalf("expected migrated main.go to import internal/runner, got:\n%s", main)
	}
}

func scaffoldLegacyBundleWorkspace(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	for _, dir := range []string{"cmd/service", "feature"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", dir, err)
		}
	}

	legacyMain := `package main

import "github.com/spf13/cobra"

func genServerProvider(host string, port int) {}
var rootCmd = &cobra.Command{}
func main() {}
`
	files := map[string]string{
		"go.work":             "go 1.25\n\nuse ./cmd/service\n",
		"cmd/service/go.mod":  "module github.com/example/legacy-repo/cmd/service\n\ngo 1.25\n",
		"cmd/service/main.go": legacyMain,
		"cmd/service/features.go": `package main

import (
	_ "github.com/example/legacy-repo/feature/auth"
)
`,
		"cmd/service/middleware.go": "package main\n",
		"cmd/service/healthz.go":    "package main\n",
	}

	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", rel, err)
		}
	}

	return root
}
