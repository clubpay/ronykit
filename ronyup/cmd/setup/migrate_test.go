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
	runnerDir := filepath.Join(root, "pkg", "runner")
	bundleDir := filepath.Join(root, "cmd", defaultBundleName)

	for _, dir := range []string{runnerDir, bundleDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
	}

	currentMain := `package main

import "github.com/example/app/pkg/runner"

func main() {
	runner.Execute(runner.Config{})
}
`
	files := map[string]string{
		filepath.Join(runnerDir, "runner.go"): "package runner\n",
		filepath.Join(runnerDir, "go.mod"):    "module github.com/example/app/pkg/runner\n\ngo 1.25\n",
		filepath.Join(bundleDir, "main.go"):   currentMain,
		bundlesManifestPath(root):             "bundles:\n  all-in-one:\n    services: [\"*\"]\n",
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
		"copy pkg/runner/",
		"rewrite cmd/all-in-one/main.go",
		"remove cmd/all-in-one/middleware.go",
		"remove cmd/all-in-one/healthz.go",
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

	if fileExists(filepath.Join(root, "pkg", "runner", "runner.go")) {
		t.Fatal("dry-run should not create pkg/runner")
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
		"pkg/runner/runner.go",
		"bundles.yaml",
		"cmd/all-in-one/main.go",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s after migration: %v", rel, err)
		}
	}

	for _, rel := range []string{"cmd/all-in-one/middleware.go", "cmd/all-in-one/healthz.go", "cmd/service/middleware.go", "cmd/service/healthz.go"} {
		if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			t.Fatalf("expected %s removed after migration: %v", rel, err)
		}
	}

	main, err := os.ReadFile(filepath.Join(root, "cmd/all-in-one/main.go"))
	if err != nil {
		t.Fatalf("ReadFile main.go: %v", err)
	}

	if !strings.Contains(string(main), "pkg/runner") {
		t.Fatalf("expected migrated main.go to import pkg/runner, got:\n%s", main)
	}
}

func TestRunMigrateBundles_FromFullstackRepoRoot(t *testing.T) {
	root := scaffoldLegacyBundleWorkspace(t)
	backend := filepath.Join(root, backendDir)
	if err := os.MkdirAll(filepath.Join(backend, "cmd"), 0o755); err != nil {
		t.Fatalf("MkdirAll backend/cmd: %v", err)
	}

	for _, rel := range []string{"go.work", "cmd/service", "feature"} {
		src := filepath.Join(root, rel)
		dst := filepath.Join(backend, rel)
		if err := os.Rename(src, dst); err != nil {
			t.Fatalf("Rename %s: %v", rel, err)
		}
	}

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
	opt.RepositoryGoModule = "github.com/example/legacy-repo/backend"

	cmd := newSilentCommand(t)
	if err := runMigrateBundles(cmd); err != nil {
		t.Fatalf("runMigrateBundles(fullstack root): %v", err)
	}

	if _, err := os.Stat(filepath.Join(backend, "pkg/runner/runner.go")); err != nil {
		t.Fatalf("expected pkg/runner under backend: %v", err)
	}
}

func TestRunMigrateBundles_FromLegacyCmdRunnerLayout(t *testing.T) {
	root := t.TempDir()

	runnerDir := filepath.Join(root, "cmd", "runner")
	serviceDir := filepath.Join(root, "cmd", "service")
	for _, dir := range []string{runnerDir, serviceDir, filepath.Join(root, "feature")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
	}

	files := map[string]string{
		"go.work":              "go 1.25\n\nuse ./cmd/runner\nuse ./cmd/service\n",
		"cmd/runner/go.mod":    "module github.com/example/legacy-repo/cmd/runner\n\ngo 1.25\n",
		"cmd/runner/runner.go": "package runner\n",
		"cmd/service/go.mod":   "module github.com/example/legacy-repo/cmd/service\n\ngo 1.25\n",
		"cmd/service/main.go": `package main

import "github.com/example/legacy-repo/cmd/runner"

func main() {
	runner.Execute(runner.Config{})
}
`,
		"cmd/service/features.go": `package main

import (
	_ "github.com/example/legacy-repo/feature/auth"
)
`,
		"bundles.yaml": "bundles:\n  service:\n    services: [\"*\"]\n",
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", rel, err)
		}
	}

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

	if _, err := os.Stat(filepath.Join(root, "pkg/runner/runner.go")); err != nil {
		t.Fatalf("expected pkg/runner after migration: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "cmd/runner/runner.go")); !os.IsNotExist(err) {
		t.Fatalf("expected legacy cmd/runner removed: %v", err)
	}

	main, err := os.ReadFile(filepath.Join(root, "cmd/all-in-one/main.go"))
	if err != nil {
		t.Fatalf("ReadFile main.go: %v", err)
	}

	if !strings.Contains(string(main), "pkg/runner") {
		t.Fatalf("expected migrated main.go to import pkg/runner, got:\n%s", main)
	}

	if _, err := os.Stat(filepath.Join(root, "cmd/service")); !os.IsNotExist(err) {
		t.Fatalf("expected legacy cmd/service removed after migration: %v", err)
	}
}

func TestRunMigrateBundles_RenamesLegacyServiceBundle(t *testing.T) {
	root := t.TempDir()

	runnerDir := filepath.Join(root, "pkg", "runner")
	serviceDir := filepath.Join(root, "cmd", "service")
	for _, dir := range []string{runnerDir, serviceDir, filepath.Join(root, "feature")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
	}

	files := map[string]string{
		"go.work":              "go 1.25\n\nuse ./pkg/runner\nuse ./cmd/service\n",
		"pkg/runner/go.mod":    "module github.com/example/legacy-repo/pkg/runner\n\ngo 1.25\n",
		"pkg/runner/runner.go": "package runner\n",
		"cmd/service/go.mod":   "module github.com/example/legacy-repo/cmd/service\n\ngo 1.25\n",
		"cmd/service/main.go": `package main

import "github.com/example/legacy-repo/pkg/runner"

func main() {
	runner.Execute(runner.Config{})
}
`,
		"cmd/service/features.go": `package main

import (
	_ "github.com/example/legacy-repo/feature/auth"
)
`,
		"bundles.yaml": "bundles:\n  service:\n    services: [\"*\"]\n",
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", rel, err)
		}
	}

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

	if _, err := os.Stat(filepath.Join(root, "cmd/all-in-one/main.go")); err != nil {
		t.Fatalf("expected cmd/all-in-one after rename: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "cmd/service")); !os.IsNotExist(err) {
		t.Fatalf("expected legacy cmd/service removed: %v", err)
	}

	cfg, err := loadBundlesConfig(root)
	if err != nil {
		t.Fatalf("loadBundlesConfig(): %v", err)
	}

	if _, ok := cfg.Bundles[defaultBundleName]; !ok {
		t.Fatalf("expected %q bundle in bundles.yaml", defaultBundleName)
	}

	if _, ok := cfg.Bundles[legacyDefaultBundleName]; ok {
		t.Fatalf("expected legacy %q bundle removed from bundles.yaml", legacyDefaultBundleName)
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
