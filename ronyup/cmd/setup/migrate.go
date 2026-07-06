package setup

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/clubpay/ronykit/ronyup/internal"
	"github.com/clubpay/ronykit/ronyup/internal/z"
	"github.com/clubpay/ronykit/x/rkit"
	"github.com/spf13/cobra"
)

var migrateOpt = struct {
	DryRun bool
}{}

var CmdSetupMigrate = &cobra.Command{
	Use:   "migrate",
	Short: "Upgrade an existing workspace to a newer scaffold layout",
}

var CmdSetupMigrateBundles = &cobra.Command{
	Use:   "bundles",
	Short: "Migrate a legacy workspace to the bundle + pkg/runner layout",
	Long: `Upgrade workspaces created before executable bundles were introduced.

The command is idempotent: safe to run multiple times. It will:

  - add pkg/runner/ (shared bootstrap) when missing or outdated
  - rewrite cmd/all-in-one/main.go to delegate to pkg/runner
  - remove legacy cmd/all-in-one/middleware.go and healthz.go (or the same under cmd/service/)
  - rename legacy cmd/service/ to cmd/all-in-one/ when present
  - remove legacy internal/runner/ and cmd/runner/ when present
  - create bundles.yaml when missing (default all-in-one bundle uses "*")
  - register pkg/runner in go.work and refresh bundle features.go files

Run from the Go workspace root (directory with go.work) or from the repository
root in a fullstack workspace (where go.work lives under backend/).

Examples:
  ronyup setup migrate bundles
  ronyup setup migrate bundles --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}

		return runMigrateBundles(cmd)
	},
}

func init() {
	flags := CmdSetupMigrateBundles.Flags()
	flags.BoolVar(&migrateOpt.DryRun, "dry-run", false, "print planned changes without writing files")

	CmdSetupMigrate.AddCommand(CmdSetupMigrateBundles)
	Cmd.AddCommand(CmdSetupMigrate)
}

type bundleLayoutStatus struct {
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

func (s bundleLayoutStatus) NeedsMigration() bool {
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

func (s bundleLayoutStatus) IsCurrent() bool {
	return !s.NeedsMigration()
}

func detectBundleLayout(goRoot string) bundleLayoutStatus {
	status := bundleLayoutStatus{
		HasRunnerModule:         fileExists(filepath.Join(runnerDir(goRoot), "go.mod")),
		HasRunnerPackage:        fileExists(filepath.Join(runnerDir(goRoot), "runner.go")),
		HasLegacyCmdRunner:      fileExists(filepath.Join(legacyCmdRunnerDir(goRoot), "runner.go")),
		HasLegacyInternalRunner: fileExists(filepath.Join(legacyRunnerDir(goRoot), "runner.go")),
		HasBundlesYAML:          fileExists(bundlesManifestPath(goRoot)),
		HasDefaultBundle:        fileExists(filepath.Join(defaultBundleDir(goRoot), "main.go")),
		HasLegacyServiceBundle:  fileExists(filepath.Join(legacyDefaultBundleDir(goRoot), "main.go")),
	}

	if cfg, err := loadBundlesConfig(goRoot); err == nil {
		_, status.HasLegacyServiceBundleInYAML = cfg.Bundles[legacyDefaultBundleName]
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

	for _, bundleDir := range []string{defaultBundleDir(goRoot), legacyDefaultBundleDir(goRoot)} {
		if !status.LegacyMiddleware && fileExists(filepath.Join(bundleDir, "middleware.go")) {
			status.LegacyMiddleware = true
		}

		if !status.LegacyHealthz && fileExists(filepath.Join(bundleDir, "healthz.go")) {
			status.LegacyHealthz = true
		}
	}

	return status
}

func defaultBundleMainPath(goRoot string) string {
	mainPath := filepath.Join(defaultBundleDir(goRoot), "main.go")
	if fileExists(mainPath) {
		return mainPath
	}

	return filepath.Join(legacyDefaultBundleDir(goRoot), "main.go")
}

func runMigrateBundles(cmd *cobra.Command) error {
	goRoot, err := resolveGoWorkspace(rkit.GetCurrentDir())
	if err != nil {
		return err
	}

	if f := cmd.Flag("repoModule"); f == nil || !f.Changed {
		detected, err := detectGoModule(goRoot)
		if err != nil {
			return fmt.Errorf("could not auto-detect repository go module: %w", err)
		}

		opt.RepositoryGoModule = detected
	}

	cmd.Printf("Go workspace: %s\n", goRoot)

	status := detectBundleLayout(goRoot)

	cmdCtx := workspaceCommandContext{
		cmd:        cmd,
		goRoot:     goRoot,
		repoModule: opt.RepositoryGoModule,
	}

	if status.IsCurrent() {
		cmd.Println("Workspace already uses the bundle layout")

		if migrateOpt.DryRun {
			return nil
		}

		return syncAllBundleFeatures(cmdCtx)
	}

	appName, err := detectApplicationName(goRoot)
	if err != nil {
		return err
	}

	templateInput := TemplateInput{
		ApplicationName: appName,
		RepositoryPath:  strings.TrimSuffix(opt.RepositoryGoModule, "/"),
		PackageName:     appName,
		RonyKitPath:     "github.com/clubpay/ronykit",
	}

	plan := buildMigratePlan(status)
	for _, step := range plan {
		cmd.Printf("plan: %s\n", step)
	}

	if migrateOpt.DryRun {
		cmd.Println("Dry run complete — no files changed")

		return nil
	}

	if err := applyMigrateBundles(cmdCtx, status, templateInput); err != nil {
		return err
	}

	cmd.Println("Bundle layout migration complete")
	cmd.Println("Tip: run `ronyup setup sync --only backend` to refresh Makefile targets")

	return nil
}

func buildMigratePlan(status bundleLayoutStatus) []string {
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
		steps = append(steps, fmt.Sprintf("rename cmd/%s/ to cmd/%s/", legacyDefaultBundleName, defaultBundleName))
	}

	if status.HasLegacyServiceBundleInYAML {
		steps = append(steps, fmt.Sprintf("rename %q bundle to %q in bundles.yaml", legacyDefaultBundleName, defaultBundleName))
	}

	if status.LegacyMain || !status.UsesRunnerMain || status.HasLegacyCmdRunner {
		steps = append(steps, fmt.Sprintf("rewrite cmd/%s/main.go to use pkg/runner", defaultBundleName))
	}

	if status.LegacyMiddleware {
		steps = append(steps, fmt.Sprintf("remove cmd/%s/middleware.go", defaultBundleName))
	}

	if status.LegacyHealthz {
		steps = append(steps, fmt.Sprintf("remove cmd/%s/healthz.go", defaultBundleName))
	}

	if !status.HasBundlesYAML {
		steps = append(steps, "create bundles.yaml with default all-in-one bundle")
	}

	if !status.HasRunnerModule {
		steps = append(steps, "initialize pkg/runner go module and go work use")
	}

	if !status.HasDefaultBundle {
		steps = append(steps, fmt.Sprintf("initialize cmd/%s module and features.go", defaultBundleName))
	}

	steps = append(steps, "regenerate bundle features.go files")

	return steps
}

func applyMigrateBundles(
	cmdCtx workspaceCommandContext,
	status bundleLayoutStatus,
	templateInput TemplateInput,
) error {
	if err := copyRunnerScaffold(cmdCtx.goRoot, templateInput, true); err != nil {
		return err
	}

	if err := renameLegacyServiceBundle(cmdCtx); err != nil {
		return err
	}

	if err := ensureDefaultBundleModule(cmdCtx); err != nil {
		return err
	}

	if status.LegacyMain || !status.UsesRunnerMain || status.HasLegacyCmdRunner || !status.HasDefaultBundle {
		if err := backupLegacyMain(cmdCtx); err != nil {
			return err
		}

		if err := renderDefaultBundleMain(defaultBundleDir(cmdCtx.goRoot), templateInput); err != nil {
			return err
		}

		cmdCtx.cmd.Printf("Updated cmd/%s/main.go\n", defaultBundleName)
	}

	for _, rel := range []string{"middleware.go", "healthz.go"} {
		for _, bundleDir := range []string{defaultBundleDir(cmdCtx.goRoot), legacyDefaultBundleDir(cmdCtx.goRoot)} {
			path := filepath.Join(bundleDir, rel)
			if !fileExists(path) {
				continue
			}

			if err := os.Remove(path); err != nil {
				return fmt.Errorf("remove %s: %w", path, err)
			}

			cmdCtx.cmd.Printf("Removed %s\n", path)
		}
	}

	if err := removeLegacyInternalRunner(cmdCtx); err != nil {
		return err
	}

	if err := removeLegacyCmdRunner(cmdCtx); err != nil {
		return err
	}

	if !status.HasBundlesYAML {
		if err := seedBundlesManifest(cmdCtx.goRoot); err != nil {
			return err
		}

		cmdCtx.cmd.Println("Created bundles.yaml")
	}

	if err := ensureRunnerModule(cmdCtx); err != nil {
		return err
	}

	if err := tidyDefaultBundleModule(cmdCtx); err != nil {
		return err
	}

	return syncAllBundleFeatures(cmdCtx)
}

func copyRunnerScaffold(goRoot string, templateInput TemplateInput, overwrite bool) error {
	dest := runnerDir(goRoot)

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

	srcPath := filepath.Join("skeleton", "backend", "cmd", defaultBundleName, "main.gotmpl")
	destPath := filepath.Join(bundleDir, "main.go")

	return z.CopyFile(z.CopyFileParams{
		FS:             internal.Skeleton,
		SrcPath:        srcPath,
		DestPath:       destPath,
		TemplateSuffix: "tmpl",
		TemplateInput:  templateInput,
	})
}

func backupLegacyMain(cmdCtx workspaceCommandContext) error {
	mainPath := defaultBundleMainPath(cmdCtx.goRoot)
	backupPath := mainPath + ".legacy"

	content, err := os.ReadFile(mainPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}

	if fileExists(backupPath) {
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

	cmdCtx.cmd.Printf("Backed up legacy main.go to cmd/%s/main.go.legacy\n", defaultBundleName)

	return nil
}

func seedBundlesManifest(goRoot string) error {
	dest := bundlesManifestPath(goRoot)
	if fileExists(dest) {
		return nil
	}

	data, err := internal.Skeleton.ReadFile(filepath.Join("skeleton", "backend", bundlesManifestName))
	if err != nil {
		return saveBundlesConfig(goRoot, BundlesConfig{
			Bundles: map[string]BundleSpec{
				defaultBundleName: {
					Description: "All-in-one dev binary (imports every feature)",
					Services:    []string{wildcardService},
				},
			},
		})
	}

	return os.WriteFile(bundlesManifestPath(goRoot), data, 0o644)
}

func ensureRunnerModule(cmdCtx workspaceCommandContext) error {
	dir := runnerDir(cmdCtx.goRoot)
	modulePath := path.Join(opt.RepositoryGoModule, runnerRelDir)
	p := z.RunCmdParams{Dir: dir}

	if !fileExists(filepath.Join(dir, "go.mod")) {
		z.RunCmd(context.Background(), p, "go", "mod", "init", modulePath)
		z.RunCmd(context.Background(), p, "go", "mod", "edit", "-go=1.25")
		cmdCtx.cmd.Println("Initialized pkg/runner module")
	}

	z.RunCmd(context.Background(), p, "go", "mod", "tidy", "-e")
	z.RunCmd(context.Background(), p, "go", "fmt", "./...")

	workDir := z.RunCmdParams{Dir: cmdCtx.goRoot}
	z.RunCmd(context.Background(), workDir, "go", "work", "use", "./"+runnerRelDir)
	z.RunCmd(context.Background(), workDir, "go", "work", "edit", "-dropuse", "./cmd/runner")
	z.RunCmd(context.Background(), workDir, "go", "work", "edit", "-dropuse", "./internal/runner")

	return nil
}

func removeLegacyCmdRunner(cmdCtx workspaceCommandContext) error {
	legacy := legacyCmdRunnerDir(cmdCtx.goRoot)
	if !fileExists(filepath.Join(legacy, "runner.go")) {
		return nil
	}

	if err := os.RemoveAll(legacy); err != nil {
		return fmt.Errorf("remove legacy cmd/runner: %w", err)
	}

	cmdCtx.cmd.Println("Removed legacy cmd/runner/")

	workDir := z.RunCmdParams{Dir: cmdCtx.goRoot}
	z.RunCmd(context.Background(), workDir, "go", "work", "edit", "-dropuse", "./cmd/runner")

	return nil
}

func removeLegacyInternalRunner(cmdCtx workspaceCommandContext) error {
	legacy := legacyRunnerDir(cmdCtx.goRoot)
	if !fileExists(filepath.Join(legacy, "runner.go")) {
		return nil
	}

	if err := os.RemoveAll(legacy); err != nil {
		return fmt.Errorf("remove legacy internal/runner: %w", err)
	}

	cmdCtx.cmd.Println("Removed legacy internal/runner/")

	workDir := z.RunCmdParams{Dir: cmdCtx.goRoot}
	z.RunCmd(context.Background(), workDir, "go", "work", "edit", "-dropuse", "./internal/runner")

	return nil
}

func tidyDefaultBundleModule(cmdCtx workspaceCommandContext) error {
	bundleDir := defaultBundleDir(cmdCtx.goRoot)
	if !fileExists(filepath.Join(bundleDir, "go.mod")) {
		return nil
	}

	p := z.RunCmdParams{Dir: bundleDir}
	z.RunCmd(context.Background(), p, "go", "mod", "tidy", "-e")
	z.RunCmd(context.Background(), p, "go", "fmt", "./...")

	return nil
}

func ensureDefaultBundleModule(cmdCtx workspaceCommandContext) error {
	bundleDir := defaultBundleDir(cmdCtx.goRoot)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return fmt.Errorf("create cmd/%s: %w", defaultBundleName, err)
	}

	if err := ensureDefaultBundleFeaturesGo(cmdCtx); err != nil {
		return err
	}

	modulePath := path.Join(opt.RepositoryGoModule, "cmd", defaultBundleName)
	p := z.RunCmdParams{Dir: bundleDir}

	if !fileExists(filepath.Join(bundleDir, "go.mod")) {
		z.RunCmd(context.Background(), p, "go", "mod", "init", modulePath)
		z.RunCmd(context.Background(), p, "go", "mod", "edit", "-go=1.25")
		cmdCtx.cmd.Printf("Initialized cmd/%s module\n", defaultBundleName)
	}

	workDir := z.RunCmdParams{Dir: cmdCtx.goRoot}
	z.RunCmd(context.Background(), workDir, "go", "work", "use", "./cmd/"+defaultBundleName)
	z.RunCmd(context.Background(), workDir, "go", "work", "edit", "-dropuse", "./cmd/"+legacyDefaultBundleName)

	return nil
}

func ensureDefaultBundleFeaturesGo(cmdCtx workspaceCommandContext) error {
	featuresPath := filepath.Join(defaultBundleDir(cmdCtx.goRoot), "features.go")
	if fileExists(featuresPath) {
		return nil
	}

	legacyPath := filepath.Join(legacyDefaultBundleDir(cmdCtx.goRoot), "features.go")
	if fileExists(legacyPath) {
		content, err := os.ReadFile(legacyPath)
		if err != nil {
			return fmt.Errorf("read cmd/%s/features.go: %w", legacyDefaultBundleName, err)
		}

		if err := os.WriteFile(featuresPath, content, 0o644); err != nil {
			return err
		}

		cmdCtx.cmd.Printf("Copied features.go from cmd/%s/\n", legacyDefaultBundleName)

		return nil
	}

	imports, err := discoverFeatureModuleImports(cmdCtx.goRoot, cmdCtx.repoModule)
	if err != nil {
		return err
	}

	if err := os.WriteFile(featuresPath, []byte(renderFeaturesGo(imports)), 0o644); err != nil {
		return err
	}

	if len(imports) > 0 {
		cmdCtx.cmd.Printf(
			"Created cmd/%s/features.go with %d feature import(s) discovered under %s/\n",
			defaultBundleName,
			len(imports),
			opt.FeatureContainerFolder,
		)
	} else {
		cmdCtx.cmd.Printf("Created empty cmd/%s/features.go\n", defaultBundleName)
	}

	return nil
}

func renameLegacyServiceBundle(cmdCtx workspaceCommandContext) error {
	legacy := legacyDefaultBundleDir(cmdCtx.goRoot)
	current := defaultBundleDir(cmdCtx.goRoot)

	if fileExists(filepath.Join(legacy, "main.go")) {
		if fileExists(filepath.Join(current, "main.go")) {
			if err := os.RemoveAll(legacy); err != nil {
				return fmt.Errorf("remove legacy cmd/%s: %w", legacyDefaultBundleName, err)
			}

			cmdCtx.cmd.Printf("Removed legacy cmd/%s/\n", legacyDefaultBundleName)
		} else if err := os.Rename(legacy, current); err != nil {
			return fmt.Errorf("rename cmd/%s to cmd/%s: %w", legacyDefaultBundleName, defaultBundleName, err)
		} else {
			cmdCtx.cmd.Printf("Renamed cmd/%s/ to cmd/%s/\n", legacyDefaultBundleName, defaultBundleName)
		}

		goModPath := filepath.Join(current, "go.mod")
		if content, err := os.ReadFile(goModPath); err == nil {
			oldMod := path.Join(opt.RepositoryGoModule, "cmd", legacyDefaultBundleName)
			newMod := path.Join(opt.RepositoryGoModule, "cmd", defaultBundleName)
			updated := strings.ReplaceAll(string(content), oldMod, newMod)

			if err := os.WriteFile(goModPath, []byte(updated), 0o644); err != nil {
				return fmt.Errorf("update cmd/%s/go.mod: %w", defaultBundleName, err)
			}
		}

		workDir := z.RunCmdParams{Dir: cmdCtx.goRoot}
		z.RunCmd(context.Background(), workDir, "go", "work", "use", "./cmd/"+defaultBundleName)
		z.RunCmd(context.Background(), workDir, "go", "work", "edit", "-dropuse", "./cmd/"+legacyDefaultBundleName)
	}

	return migrateLegacyBundlesManifest(cmdCtx)
}

func migrateLegacyBundlesManifest(cmdCtx workspaceCommandContext) error {
	cfg, err := loadBundlesConfig(cmdCtx.goRoot)
	if err != nil {
		return err
	}

	spec, ok := cfg.Bundles[legacyDefaultBundleName]
	if !ok {
		return nil
	}

	if _, exists := cfg.Bundles[defaultBundleName]; !exists {
		cfg.Bundles[defaultBundleName] = spec
	}

	delete(cfg.Bundles, legacyDefaultBundleName)

	if err := saveBundlesConfig(cmdCtx.goRoot, cfg); err != nil {
		return err
	}

	cmdCtx.cmd.Printf(
		"Renamed %q bundle to %q in %s\n",
		legacyDefaultBundleName,
		defaultBundleName,
		bundlesManifestName,
	)

	return nil
}
