package scaffold

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/clubpay/ronykit/ronyup/internal"
	"github.com/clubpay/ronykit/ronyup/internal/z"
)

// MigrateBundlesRequest configures MigrateBundles.
type MigrateBundlesRequest struct {
	// StartDir is where Go workspace resolution begins (workspace root or the
	// repository root in a fullstack workspace).
	StartDir string
	// Module overrides auto-detection from go.work when non-empty.
	Module string
	// AppName overrides the application name used in templates when non-empty.
	AppName string
	// DryRun prints the planned changes without writing files.
	DryRun bool
}

// BundleLayoutStatus captures which parts of the bundle + pkg/runner layout
// are present in a workspace.
type BundleLayoutStatus struct {
	HasRunnerModule              bool
	HasRunnerPackage             bool
	HasLegacyCmdRunner           bool
	HasLegacyInternalRunner      bool
	HasBundlesYAML               bool
	HasDefaultBundle             bool
	HasLegacyServiceBundle       bool
	HasLegacyServiceBundleInYAML bool
	UsesRunnerMain               bool
	LegacyMain                   bool
	LegacyMiddleware             bool
	LegacyHealthz                bool
}

// NeedsMigration reports whether any migration step is required.
func (s BundleLayoutStatus) NeedsMigration() bool {
	if s.LegacyMiddleware || s.LegacyHealthz || s.LegacyMain {
		return true
	}

	if s.HasLegacyInternalRunner || s.HasLegacyCmdRunner {
		return true
	}

	if s.HasLegacyServiceBundle && !s.HasDefaultBundle {
		return true
	}

	if s.HasLegacyServiceBundleInYAML {
		return true
	}

	if !s.HasRunnerPackage || !s.HasBundlesYAML || !s.UsesRunnerMain {
		return true
	}

	return false
}

// IsCurrent reports whether the workspace already uses the bundle layout.
func (s BundleLayoutStatus) IsCurrent() bool {
	return !s.NeedsMigration()
}

// DetectBundleLayout inspects goRoot and reports the bundle layout status.
func DetectBundleLayout(goRoot string) BundleLayoutStatus {
	return detectBundleLayout(goRoot)
}

func detectBundleLayout(goRoot string) BundleLayoutStatus {
	status := BundleLayoutStatus{
		HasRunnerModule:         FileExists(filepath.Join(RunnerDir(goRoot), "go.mod")),
		HasRunnerPackage:        FileExists(filepath.Join(RunnerDir(goRoot), "runner.go")),
		HasLegacyCmdRunner:      FileExists(filepath.Join(LegacyCmdRunnerDir(goRoot), "runner.go")),
		HasLegacyInternalRunner: FileExists(filepath.Join(LegacyRunnerDir(goRoot), "runner.go")),
		HasBundlesYAML:          FileExists(BundlesManifestPath(goRoot)),
		HasDefaultBundle:        FileExists(filepath.Join(DefaultBundleDir(goRoot), "main.go")),
		HasLegacyServiceBundle:  FileExists(filepath.Join(LegacyDefaultBundleDir(goRoot), "main.go")),
	}

	if cfg, err := LoadBundlesConfig(goRoot); err == nil {
		_, status.HasLegacyServiceBundleInYAML = cfg.Bundles[LegacyDefaultBundleName]
	}

	mainPath := defaultBundleMainPath(goRoot)
	if content, err := os.ReadFile(mainPath); err == nil {
		text := string(content)
		status.UsesRunnerMain = (strings.Contains(text, "/pkg/runner") ||
			strings.Contains(text, "/cmd/runner") ||
			strings.Contains(text, "/internal/runner")) &&
			strings.Contains(text, "runner.Execute")
		status.LegacyMain = strings.Contains(text, "genServerProvider") ||
			strings.Contains(text, "newRootCommand") ||
			(strings.Contains(text, "cobra.Command") && !status.UsesRunnerMain)
	}

	for _, bundleDir := range []string{DefaultBundleDir(goRoot), LegacyDefaultBundleDir(goRoot)} {
		if !status.LegacyMiddleware && FileExists(filepath.Join(bundleDir, "middleware.go")) {
			status.LegacyMiddleware = true
		}

		if !status.LegacyHealthz && FileExists(filepath.Join(bundleDir, "healthz.go")) {
			status.LegacyHealthz = true
		}
	}

	return status
}

func defaultBundleMainPath(goRoot string) string {
	mainPath := filepath.Join(DefaultBundleDir(goRoot), "main.go")
	if FileExists(mainPath) {
		return mainPath
	}

	return filepath.Join(LegacyDefaultBundleDir(goRoot), "main.go")
}

// MigrateBundles upgrades a workspace created before executable bundles to the
// current layout (pkg/runner, bundles.yaml, thin cmd/all-in-one/main.go).
// The operation is idempotent.
func MigrateBundles(ctx context.Context, req MigrateBundlesRequest, log Logger) error {
	if log == nil {
		log = DiscardLogger{}
	}

	goRoot, err := resolveGoWorkspace(req.StartDir)
	if err != nil {
		return err
	}

	module := req.Module
	if module == "" {
		module, err = detectGoModule(goRoot)
		if err != nil {
			return fmt.Errorf("could not auto-detect repository go module: %w", err)
		}
	}

	log.Printf("Go workspace: %s\n", goRoot)

	status := detectBundleLayout(goRoot)

	cmdCtx := WorkspaceContext{
		Log:        log,
		GoRoot:     goRoot,
		RepoModule: module,
	}

	if status.IsCurrent() {
		log.Println("Workspace already uses the bundle layout")

		if req.DryRun {
			return nil
		}

		return SyncAllBundleFeatures(cmdCtx)
	}

	appName, err := detectApplicationName(goRoot, req.AppName)
	if err != nil {
		return err
	}

	templateInput := TemplateInput{
		ApplicationName: appName,
		RepositoryPath:  strings.TrimSuffix(module, "/"),
		PackageName:     appName,
		RonyKitPath:     RonyKitModulePath,
	}

	plan := buildMigratePlan(status)
	for _, step := range plan {
		log.Printf("plan: %s\n", step)
	}

	if req.DryRun {
		log.Println("Dry run complete — no files changed")

		return nil
	}

	if err := applyMigrateBundles(ctx, cmdCtx, status, templateInput); err != nil {
		return err
	}

	log.Println("Bundle layout migration complete")
	log.Println("Tip: run `ronyup setup sync --only backend` to refresh Makefile targets")

	return nil
}

// BuildMigratePlan returns the human-readable steps required for status.
func BuildMigratePlan(status BundleLayoutStatus) []string {
	return buildMigratePlan(status)
}

func buildMigratePlan(status BundleLayoutStatus) []string {
	var steps []string

	if !status.HasRunnerPackage {
		steps = append(steps, "copy pkg/runner/ from scaffold")
	} else if status.LegacyMiddleware || status.LegacyHealthz || status.LegacyMain {
		steps = append(steps, "refresh pkg/runner/ scaffold files")
	}

	if status.HasLegacyInternalRunner {
		steps = append(steps, "remove legacy internal/runner/")
	}

	if status.HasLegacyCmdRunner {
		steps = append(steps, "remove legacy cmd/runner/")
	}

	if status.HasLegacyServiceBundle && !status.HasDefaultBundle {
		steps = append(
			steps,
			fmt.Sprintf("rename cmd/%s/ to cmd/%s/", LegacyDefaultBundleName, DefaultBundleName),
		)
	}

	if status.HasLegacyServiceBundleInYAML {
		steps = append(
			steps,
			fmt.Sprintf("rename %q bundle to %q in bundles.yaml", LegacyDefaultBundleName, DefaultBundleName),
		)
	}

	if status.LegacyMain || !status.UsesRunnerMain || status.HasLegacyCmdRunner {
		steps = append(steps, fmt.Sprintf("rewrite cmd/%s/main.go to use pkg/runner", DefaultBundleName))
	}

	if status.LegacyMiddleware {
		steps = append(steps, fmt.Sprintf("remove cmd/%s/middleware.go", DefaultBundleName))
	}

	if status.LegacyHealthz {
		steps = append(steps, fmt.Sprintf("remove cmd/%s/healthz.go", DefaultBundleName))
	}

	if !status.HasBundlesYAML {
		steps = append(steps, "create bundles.yaml with default all-in-one bundle")
	}

	if !status.HasRunnerModule {
		steps = append(steps, "initialize pkg/runner go module and go work use")
	}

	if !status.HasDefaultBundle {
		steps = append(steps, fmt.Sprintf("initialize cmd/%s module and features.go", DefaultBundleName))
	}

	steps = append(steps, "regenerate bundle features.go files")

	return steps
}

func applyMigrateBundles(
	ctx context.Context,
	cmdCtx WorkspaceContext,
	status BundleLayoutStatus,
	templateInput TemplateInput,
) error {
	if err := copyRunnerScaffold(cmdCtx.GoRoot, templateInput, true); err != nil {
		return err
	}

	if err := renameLegacyServiceBundle(ctx, cmdCtx); err != nil {
		return err
	}

	if err := ensureDefaultBundleModule(ctx, cmdCtx); err != nil {
		return err
	}

	if status.LegacyMain || !status.UsesRunnerMain || status.HasLegacyCmdRunner || !status.HasDefaultBundle {
		if err := backupLegacyMain(cmdCtx); err != nil {
			return err
		}

		if err := renderDefaultBundleMain(DefaultBundleDir(cmdCtx.GoRoot), templateInput); err != nil {
			return err
		}

		cmdCtx.Log.Printf("Updated cmd/%s/main.go\n", DefaultBundleName)
	}

	for _, rel := range []string{"middleware.go", "healthz.go"} {
		for _, bundleDir := range []string{DefaultBundleDir(cmdCtx.GoRoot), LegacyDefaultBundleDir(cmdCtx.GoRoot)} {
			path := filepath.Join(bundleDir, rel)
			if !FileExists(path) {
				continue
			}

			if err := os.Remove(path); err != nil {
				return fmt.Errorf("remove %s: %w", path, err)
			}

			cmdCtx.Log.Printf("Removed %s\n", path)
		}
	}

	if err := removeLegacyInternalRunner(ctx, cmdCtx); err != nil {
		return err
	}

	if err := removeLegacyCmdRunner(ctx, cmdCtx); err != nil {
		return err
	}

	if !status.HasBundlesYAML {
		if err := seedBundlesManifest(cmdCtx.GoRoot); err != nil {
			return err
		}

		cmdCtx.Log.Println("Created bundles.yaml")
	}

	if err := ensureRunnerModule(ctx, cmdCtx); err != nil {
		return err
	}

	if err := tidyDefaultBundleModule(ctx, cmdCtx); err != nil {
		return err
	}

	return SyncAllBundleFeatures(cmdCtx)
}

func copyRunnerScaffold(goRoot string, templateInput TemplateInput, overwrite bool) error {
	dest := RunnerDir(goRoot)

	return z.CopyDir(z.CopyDirParams{
		FS:             internal.Skeleton,
		SrcPathPrefix:  filepath.Join("skeleton", "backend", "pkg", "runner"),
		DestPathPrefix: dest,
		TemplateInput:  templateInput,
		SkipExisting:   !overwrite,
	})
}

func renderDefaultBundleMain(bundleDir string, templateInput TemplateInput) error {
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", bundleDir, err)
	}

	srcPath := filepath.Join("skeleton", "backend", "cmd", DefaultBundleName, "main.gotmpl")
	destPath := filepath.Join(bundleDir, "main.go")

	return z.CopyFile(z.CopyFileParams{
		FS:             internal.Skeleton,
		SrcPath:        srcPath,
		DestPath:       destPath,
		TemplateSuffix: "tmpl",
		TemplateInput:  templateInput,
	})
}

func backupLegacyMain(cmdCtx WorkspaceContext) error {
	mainPath := defaultBundleMainPath(cmdCtx.GoRoot)
	backupPath := mainPath + ".legacy"

	content, err := os.ReadFile(mainPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}

	if FileExists(backupPath) {
		return nil
	}

	if strings.Contains(string(content), "/pkg/runner") ||
		strings.Contains(string(content), "/cmd/runner") ||
		strings.Contains(string(content), "/internal/runner") {
		return nil
	}

	if err := os.WriteFile(backupPath, content, 0o644); err != nil {
		return fmt.Errorf("backup legacy main.go: %w", err)
	}

	cmdCtx.Log.Printf("Backed up legacy main.go to cmd/%s/main.go.legacy\n", DefaultBundleName)

	return nil
}

func seedBundlesManifest(goRoot string) error {
	dest := BundlesManifestPath(goRoot)
	if FileExists(dest) {
		return nil
	}

	data, err := internal.Skeleton.ReadFile(filepath.Join("skeleton", "backend", BundlesManifestName))
	if err != nil {
		return SaveBundlesConfig(goRoot, BundlesConfig{
			Bundles: map[string]BundleSpec{
				DefaultBundleName: {
					Description: "All-in-one dev binary (imports every feature)",
					Services:    []string{WildcardService},
				},
			},
		})
	}

	return os.WriteFile(BundlesManifestPath(goRoot), data, 0o644)
}

func ensureRunnerModule(ctx context.Context, cmdCtx WorkspaceContext) error {
	dir := RunnerDir(cmdCtx.GoRoot)
	modulePath := path.Join(cmdCtx.RepoModule, RunnerRelDir)
	p := z.RunCmdParams{Dir: dir}

	if !FileExists(filepath.Join(dir, "go.mod")) {
		if err := z.RunCmd(ctx, p, "go", "mod", "init", modulePath); err != nil {
			return err
		}

		if err := z.RunCmd(ctx, p, "go", "mod", "edit", "-go=1.25"); err != nil {
			return err
		}

		cmdCtx.Log.Println("Initialized pkg/runner module")
	}

	if err := z.RunCmd(ctx, p, "go", "mod", "tidy", "-e"); err != nil {
		return err
	}

	if err := z.RunCmd(ctx, p, "go", "fmt", "./..."); err != nil {
		return err
	}

	workDir := z.RunCmdParams{Dir: cmdCtx.GoRoot}
	if err := z.RunCmd(ctx, workDir, "go", "work", "use", "./"+RunnerRelDir); err != nil {
		return err
	}
	// Best-effort cleanup of legacy use directives.
	_ = z.RunCmd(ctx, workDir, "go", "work", "edit", "-dropuse", "./cmd/runner")
	_ = z.RunCmd(ctx, workDir, "go", "work", "edit", "-dropuse", "./internal/runner")

	return nil
}

func removeLegacyCmdRunner(ctx context.Context, cmdCtx WorkspaceContext) error {
	legacy := LegacyCmdRunnerDir(cmdCtx.GoRoot)
	if !FileExists(filepath.Join(legacy, "runner.go")) {
		return nil
	}

	if err := os.RemoveAll(legacy); err != nil {
		return fmt.Errorf("remove legacy cmd/runner: %w", err)
	}

	cmdCtx.Log.Println("Removed legacy cmd/runner/")

	workDir := z.RunCmdParams{Dir: cmdCtx.GoRoot}
	_ = z.RunCmd(ctx, workDir, "go", "work", "edit", "-dropuse", "./cmd/runner")

	return nil
}

func removeLegacyInternalRunner(ctx context.Context, cmdCtx WorkspaceContext) error {
	legacy := LegacyRunnerDir(cmdCtx.GoRoot)
	if !FileExists(filepath.Join(legacy, "runner.go")) {
		return nil
	}

	if err := os.RemoveAll(legacy); err != nil {
		return fmt.Errorf("remove legacy internal/runner: %w", err)
	}

	cmdCtx.Log.Println("Removed legacy internal/runner/")

	workDir := z.RunCmdParams{Dir: cmdCtx.GoRoot}
	_ = z.RunCmd(ctx, workDir, "go", "work", "edit", "-dropuse", "./internal/runner")

	return nil
}

func tidyDefaultBundleModule(ctx context.Context, cmdCtx WorkspaceContext) error {
	bundleDir := DefaultBundleDir(cmdCtx.GoRoot)
	if !FileExists(filepath.Join(bundleDir, "go.mod")) {
		return nil
	}

	p := z.RunCmdParams{Dir: bundleDir}
	if err := z.RunCmd(ctx, p, "go", "mod", "tidy", "-e"); err != nil {
		return err
	}

	if err := z.RunCmd(ctx, p, "go", "fmt", "./..."); err != nil {
		return err
	}

	return nil
}

func ensureDefaultBundleModule(ctx context.Context, cmdCtx WorkspaceContext) error {
	bundleDir := DefaultBundleDir(cmdCtx.GoRoot)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return fmt.Errorf("create cmd/%s: %w", DefaultBundleName, err)
	}

	if err := ensureDefaultBundleFeaturesGo(cmdCtx); err != nil {
		return err
	}

	modulePath := path.Join(cmdCtx.RepoModule, "cmd", DefaultBundleName)
	p := z.RunCmdParams{Dir: bundleDir}

	if !FileExists(filepath.Join(bundleDir, "go.mod")) {
		if err := z.RunCmd(ctx, p, "go", "mod", "init", modulePath); err != nil {
			return err
		}

		if err := z.RunCmd(ctx, p, "go", "mod", "edit", "-go=1.25"); err != nil {
			return err
		}

		cmdCtx.Log.Printf("Initialized cmd/%s module\n", DefaultBundleName)
	}

	workDir := z.RunCmdParams{Dir: cmdCtx.GoRoot}
	if err := z.RunCmd(ctx, workDir, "go", "work", "use", "./cmd/"+DefaultBundleName); err != nil {
		return err
	}

	_ = z.RunCmd(ctx, workDir, "go", "work", "edit", "-dropuse", "./cmd/"+LegacyDefaultBundleName)

	return nil
}

func ensureDefaultBundleFeaturesGo(cmdCtx WorkspaceContext) error {
	featuresPath := filepath.Join(DefaultBundleDir(cmdCtx.GoRoot), "features.go")
	if FileExists(featuresPath) {
		return nil
	}

	legacyPath := filepath.Join(LegacyDefaultBundleDir(cmdCtx.GoRoot), "features.go")
	if FileExists(legacyPath) {
		content, err := os.ReadFile(legacyPath)
		if err != nil {
			return fmt.Errorf("read cmd/%s/features.go: %w", LegacyDefaultBundleName, err)
		}

		if err := os.WriteFile(featuresPath, content, 0o644); err != nil {
			return err
		}

		cmdCtx.Log.Printf("Copied features.go from cmd/%s/\n", LegacyDefaultBundleName)

		return nil
	}

	imports, err := DiscoverFeatureModuleImports(cmdCtx.GoRoot, cmdCtx.RepoModule, DefaultFeaturePrefix)
	if err != nil {
		return err
	}

	if err := os.WriteFile(featuresPath, []byte(RenderFeaturesGo(imports)), 0o644); err != nil {
		return err
	}

	if len(imports) > 0 {
		cmdCtx.Log.Printf(
			"Created cmd/%s/features.go with %d feature import(s) discovered under %s/\n",
			DefaultBundleName,
			len(imports),
			DefaultFeaturePrefix,
		)
	} else {
		cmdCtx.Log.Printf("Created empty cmd/%s/features.go\n", DefaultBundleName)
	}

	return nil
}

func renameLegacyServiceBundle(ctx context.Context, cmdCtx WorkspaceContext) error {
	legacy := LegacyDefaultBundleDir(cmdCtx.GoRoot)
	current := DefaultBundleDir(cmdCtx.GoRoot)

	if FileExists(filepath.Join(legacy, "main.go")) {
		if FileExists(filepath.Join(current, "main.go")) {
			if err := os.RemoveAll(legacy); err != nil {
				return fmt.Errorf("remove legacy cmd/%s: %w", LegacyDefaultBundleName, err)
			}

			cmdCtx.Log.Printf("Removed legacy cmd/%s/\n", LegacyDefaultBundleName)
		} else if err := os.Rename(legacy, current); err != nil {
			return fmt.Errorf("rename cmd/%s to cmd/%s: %w", LegacyDefaultBundleName, DefaultBundleName, err)
		} else {
			cmdCtx.Log.Printf("Renamed cmd/%s/ to cmd/%s/\n", LegacyDefaultBundleName, DefaultBundleName)
		}

		goModPath := filepath.Join(current, "go.mod")
		if content, err := os.ReadFile(goModPath); err == nil {
			oldMod := path.Join(cmdCtx.RepoModule, "cmd", LegacyDefaultBundleName)
			newMod := path.Join(cmdCtx.RepoModule, "cmd", DefaultBundleName)
			updated := strings.ReplaceAll(string(content), oldMod, newMod)

			if err := os.WriteFile(goModPath, []byte(updated), 0o644); err != nil {
				return fmt.Errorf("update cmd/%s/go.mod: %w", DefaultBundleName, err)
			}
		}

		workDir := z.RunCmdParams{Dir: cmdCtx.GoRoot}
		if err := z.RunCmd(ctx, workDir, "go", "work", "use", "./cmd/"+DefaultBundleName); err != nil {
			return err
		}

		_ = z.RunCmd(ctx, workDir, "go", "work", "edit", "-dropuse", "./cmd/"+LegacyDefaultBundleName)
	}

	return migrateLegacyBundlesManifest(cmdCtx)
}

func migrateLegacyBundlesManifest(cmdCtx WorkspaceContext) error {
	cfg, err := LoadBundlesConfig(cmdCtx.GoRoot)
	if err != nil {
		return err
	}

	spec, ok := cfg.Bundles[LegacyDefaultBundleName]
	if !ok {
		return nil
	}

	if _, exists := cfg.Bundles[DefaultBundleName]; !exists {
		cfg.Bundles[DefaultBundleName] = spec
	}

	delete(cfg.Bundles, LegacyDefaultBundleName)

	if err := SaveBundlesConfig(cmdCtx.GoRoot, cfg); err != nil {
		return err
	}

	cmdCtx.Log.Printf(
		"Renamed %q bundle to %q in %s\n",
		LegacyDefaultBundleName,
		DefaultBundleName,
		BundlesManifestName,
	)

	return nil
}

// detectApplicationName derives the app name from the override, the Go module
// path, or the workspace directory name.
func detectApplicationName(goRoot, override string) (string, error) {
	if override != "" {
		return override, nil
	}

	module, err := detectGoModule(goRoot)
	if err != nil {
		return "", err
	}

	base := path.Base(module)
	if base == "" || base == "." {
		base = path.Base(goRoot)
	}

	return base, nil
}
