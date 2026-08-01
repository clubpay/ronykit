package setup

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/clubpay/ronykit/ronyup/internal"
	"github.com/clubpay/ronykit/ronyup/internal/scaffold"
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
		HasRunnerModule:         scaffold.FileExists(filepath.Join(scaffold.RunnerDir(goRoot), "go.mod")),
		HasRunnerPackage:        scaffold.FileExists(filepath.Join(scaffold.RunnerDir(goRoot), "runner.go")),
		HasLegacyCmdRunner:      scaffold.FileExists(filepath.Join(scaffold.LegacyCmdRunnerDir(goRoot), "runner.go")),
		HasLegacyInternalRunner: scaffold.FileExists(filepath.Join(scaffold.LegacyRunnerDir(goRoot), "runner.go")),
		HasBundlesYAML:          scaffold.FileExists(scaffold.BundlesManifestPath(goRoot)),
		HasDefaultBundle:        scaffold.FileExists(filepath.Join(scaffold.DefaultBundleDir(goRoot), "main.go")),
		HasLegacyServiceBundle:  scaffold.FileExists(filepath.Join(scaffold.LegacyDefaultBundleDir(goRoot), "main.go")),
	}

	if cfg, err := scaffold.LoadBundlesConfig(goRoot); err == nil {
		_, status.HasLegacyServiceBundleInYAML = cfg.Bundles[scaffold.LegacyDefaultBundleName]
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

	for _, bundleDir := range []string{scaffold.DefaultBundleDir(goRoot), scaffold.LegacyDefaultBundleDir(goRoot)} {
		if !status.LegacyMiddleware && scaffold.FileExists(filepath.Join(bundleDir, "middleware.go")) {
			status.LegacyMiddleware = true
		}

		if !status.LegacyHealthz && scaffold.FileExists(filepath.Join(bundleDir, "healthz.go")) {
			status.LegacyHealthz = true
		}
	}

	return status
}

func defaultBundleMainPath(goRoot string) string {
	mainPath := filepath.Join(scaffold.DefaultBundleDir(goRoot), "main.go")
	if scaffold.FileExists(mainPath) {
		return mainPath
	}

	return filepath.Join(scaffold.LegacyDefaultBundleDir(goRoot), "main.go")
}

func runMigrateBundles(cmd *cobra.Command) error {
	goRoot, err := scaffold.ResolveGoWorkspace(rkit.GetCurrentDir())
	if err != nil {
		return err
	}

	if f := cmd.Flag("repoModule"); f == nil || !f.Changed {
		detected, err := scaffold.DetectGoModule(goRoot)
		if err != nil {
			return fmt.Errorf("could not auto-detect repository go module: %w", err)
		}

		opt.RepositoryGoModule = detected
	}

	cmd.Printf("Go workspace: %s\n", goRoot)

	status := detectBundleLayout(goRoot)

	cmdCtx := scaffold.WorkspaceContext{
		Log:        cmd,
		GoRoot:     goRoot,
		RepoModule: opt.RepositoryGoModule,
	}

	if status.IsCurrent() {
		cmd.Println("Workspace already uses the bundle layout")

		if migrateOpt.DryRun {
			return nil
		}

		return scaffold.SyncAllBundleFeatures(cmdCtx)
	}

	appName, err := detectApplicationName(goRoot)
	if err != nil {
		return err
	}

	templateInput := scaffold.TemplateInput{
		ApplicationName: appName,
		RepositoryPath:  strings.TrimSuffix(opt.RepositoryGoModule, "/"),
		PackageName:     appName,
		RonyKitPath:     scaffold.RonyKitModulePath,
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
		steps = append(
			steps,
			fmt.Sprintf("rename cmd/%s/ to cmd/%s/", scaffold.LegacyDefaultBundleName, scaffold.DefaultBundleName),
		)
	}

	if status.HasLegacyServiceBundleInYAML {
		steps = append(
			steps,
			fmt.Sprintf("rename %q bundle to %q in bundles.yaml", scaffold.LegacyDefaultBundleName, scaffold.DefaultBundleName),
		)
	}

	if status.LegacyMain || !status.UsesRunnerMain || status.HasLegacyCmdRunner {
		steps = append(steps, fmt.Sprintf("rewrite cmd/%s/main.go to use pkg/runner", scaffold.DefaultBundleName))
	}

	if status.LegacyMiddleware {
		steps = append(steps, fmt.Sprintf("remove cmd/%s/middleware.go", scaffold.DefaultBundleName))
	}

	if status.LegacyHealthz {
		steps = append(steps, fmt.Sprintf("remove cmd/%s/healthz.go", scaffold.DefaultBundleName))
	}

	if !status.HasBundlesYAML {
		steps = append(steps, "create bundles.yaml with default all-in-one bundle")
	}

	if !status.HasRunnerModule {
		steps = append(steps, "initialize pkg/runner go module and go work use")
	}

	if !status.HasDefaultBundle {
		steps = append(steps, fmt.Sprintf("initialize cmd/%s module and features.go", scaffold.DefaultBundleName))
	}

	steps = append(steps, "regenerate bundle features.go files")

	return steps
}

func applyMigrateBundles(
	cmdCtx scaffold.WorkspaceContext,
	status bundleLayoutStatus,
	templateInput TemplateInput,
) error {
	if err := copyRunnerScaffold(cmdCtx.GoRoot, templateInput, true); err != nil {
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

		if err := renderDefaultBundleMain(scaffold.DefaultBundleDir(cmdCtx.GoRoot), templateInput); err != nil {
			return err
		}

		cmdCtx.Log.Printf("Updated cmd/%s/main.go\n", scaffold.DefaultBundleName)
	}

	for _, rel := range []string{"middleware.go", "healthz.go"} {
		for _, bundleDir := range []string{scaffold.DefaultBundleDir(cmdCtx.GoRoot), scaffold.LegacyDefaultBundleDir(cmdCtx.GoRoot)} {
			path := filepath.Join(bundleDir, rel)
			if !scaffold.FileExists(path) {
				continue
			}

			if err := os.Remove(path); err != nil {
				return fmt.Errorf("remove %s: %w", path, err)
			}

			cmdCtx.Log.Printf("Removed %s\n", path)
		}
	}

	if err := removeLegacyInternalRunner(cmdCtx); err != nil {
		return err
	}

	if err := removeLegacyCmdRunner(cmdCtx); err != nil {
		return err
	}

	if !status.HasBundlesYAML {
		if err := seedBundlesManifest(cmdCtx.GoRoot); err != nil {
			return err
		}

		cmdCtx.Log.Println("Created bundles.yaml")
	}

	if err := ensureRunnerModule(cmdCtx); err != nil {
		return err
	}

	if err := tidyDefaultBundleModule(cmdCtx); err != nil {
		return err
	}

	return scaffold.SyncAllBundleFeatures(cmdCtx)
}

func copyRunnerScaffold(goRoot string, templateInput TemplateInput, overwrite bool) error {
	dest := scaffold.RunnerDir(goRoot)

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

	srcPath := filepath.Join("skeleton", "backend", "cmd", scaffold.DefaultBundleName, "main.gotmpl")
	destPath := filepath.Join(bundleDir, "main.go")

	return z.CopyFile(z.CopyFileParams{
		FS:             internal.Skeleton,
		SrcPath:        srcPath,
		DestPath:       destPath,
		TemplateSuffix: "tmpl",
		TemplateInput:  templateInput,
	})
}

func backupLegacyMain(cmdCtx scaffold.WorkspaceContext) error {
	mainPath := defaultBundleMainPath(cmdCtx.GoRoot)
	backupPath := mainPath + ".legacy"

	content, err := os.ReadFile(mainPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}

	if scaffold.FileExists(backupPath) {
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

	cmdCtx.Log.Printf("Backed up legacy main.go to cmd/%s/main.go.legacy\n", scaffold.DefaultBundleName)

	return nil
}

func seedBundlesManifest(goRoot string) error {
	dest := scaffold.BundlesManifestPath(goRoot)
	if scaffold.FileExists(dest) {
		return nil
	}

	data, err := internal.Skeleton.ReadFile(filepath.Join("skeleton", "backend", scaffold.BundlesManifestName))
	if err != nil {
		return scaffold.SaveBundlesConfig(goRoot, scaffold.BundlesConfig{
			Bundles: map[string]scaffold.BundleSpec{
				scaffold.DefaultBundleName: {
					Description: "All-in-one dev binary (imports every feature)",
					Services:    []string{scaffold.WildcardService},
				},
			},
		})
	}

	return os.WriteFile(scaffold.BundlesManifestPath(goRoot), data, 0o644)
}

func ensureRunnerModule(cmdCtx scaffold.WorkspaceContext) error {
	dir := scaffold.RunnerDir(cmdCtx.GoRoot)
	modulePath := path.Join(opt.RepositoryGoModule, scaffold.RunnerRelDir)
	p := z.RunCmdParams{Dir: dir}

	if !scaffold.FileExists(filepath.Join(dir, "go.mod")) {
		if err := z.RunCmd(context.Background(), p, "go", "mod", "init", modulePath); err != nil {
			return err
		}

		if err := z.RunCmd(context.Background(), p, "go", "mod", "edit", "-go=1.25"); err != nil {
			return err
		}

		cmdCtx.Log.Println("Initialized pkg/runner module")
	}

	if err := z.RunCmd(context.Background(), p, "go", "mod", "tidy", "-e"); err != nil {
		return err
	}

	if err := z.RunCmd(context.Background(), p, "go", "fmt", "./..."); err != nil {
		return err
	}

	workDir := z.RunCmdParams{Dir: cmdCtx.GoRoot}
	if err := z.RunCmd(context.Background(), workDir, "go", "work", "use", "./"+scaffold.RunnerRelDir); err != nil {
		return err
	}
	// Best-effort cleanup of legacy use directives.
	_ = z.RunCmd(context.Background(), workDir, "go", "work", "edit", "-dropuse", "./cmd/runner")
	_ = z.RunCmd(context.Background(), workDir, "go", "work", "edit", "-dropuse", "./internal/runner")

	return nil
}

func removeLegacyCmdRunner(cmdCtx scaffold.WorkspaceContext) error {
	legacy := scaffold.LegacyCmdRunnerDir(cmdCtx.GoRoot)
	if !scaffold.FileExists(filepath.Join(legacy, "runner.go")) {
		return nil
	}

	if err := os.RemoveAll(legacy); err != nil {
		return fmt.Errorf("remove legacy cmd/runner: %w", err)
	}

	cmdCtx.Log.Println("Removed legacy cmd/runner/")

	workDir := z.RunCmdParams{Dir: cmdCtx.GoRoot}
	_ = z.RunCmd(context.Background(), workDir, "go", "work", "edit", "-dropuse", "./cmd/runner")

	return nil
}

func removeLegacyInternalRunner(cmdCtx scaffold.WorkspaceContext) error {
	legacy := scaffold.LegacyRunnerDir(cmdCtx.GoRoot)
	if !scaffold.FileExists(filepath.Join(legacy, "runner.go")) {
		return nil
	}

	if err := os.RemoveAll(legacy); err != nil {
		return fmt.Errorf("remove legacy internal/runner: %w", err)
	}

	cmdCtx.Log.Println("Removed legacy internal/runner/")

	workDir := z.RunCmdParams{Dir: cmdCtx.GoRoot}
	_ = z.RunCmd(context.Background(), workDir, "go", "work", "edit", "-dropuse", "./internal/runner")

	return nil
}

func tidyDefaultBundleModule(cmdCtx scaffold.WorkspaceContext) error {
	bundleDir := scaffold.DefaultBundleDir(cmdCtx.GoRoot)
	if !scaffold.FileExists(filepath.Join(bundleDir, "go.mod")) {
		return nil
	}

	p := z.RunCmdParams{Dir: bundleDir}
	if err := z.RunCmd(context.Background(), p, "go", "mod", "tidy", "-e"); err != nil {
		return err
	}

	if err := z.RunCmd(context.Background(), p, "go", "fmt", "./..."); err != nil {
		return err
	}

	return nil
}

func ensureDefaultBundleModule(cmdCtx scaffold.WorkspaceContext) error {
	bundleDir := scaffold.DefaultBundleDir(cmdCtx.GoRoot)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return fmt.Errorf("create cmd/%s: %w", scaffold.DefaultBundleName, err)
	}

	if err := ensureDefaultBundleFeaturesGo(cmdCtx); err != nil {
		return err
	}

	modulePath := path.Join(opt.RepositoryGoModule, "cmd", scaffold.DefaultBundleName)
	p := z.RunCmdParams{Dir: bundleDir}

	if !scaffold.FileExists(filepath.Join(bundleDir, "go.mod")) {
		if err := z.RunCmd(context.Background(), p, "go", "mod", "init", modulePath); err != nil {
			return err
		}

		if err := z.RunCmd(context.Background(), p, "go", "mod", "edit", "-go=1.25"); err != nil {
			return err
		}

		cmdCtx.Log.Printf("Initialized cmd/%s module\n", scaffold.DefaultBundleName)
	}

	workDir := z.RunCmdParams{Dir: cmdCtx.GoRoot}
	if err := z.RunCmd(
		context.Background(),
		workDir,
		"go",
		"work",
		"use",
		"./cmd/"+scaffold.DefaultBundleName,
	); err != nil {
		return err
	}

	_ = z.RunCmd(
		context.Background(),
		workDir,
		"go",
		"work",
		"edit",
		"-dropuse",
		"./cmd/"+scaffold.LegacyDefaultBundleName,
	)

	return nil
}

func ensureDefaultBundleFeaturesGo(cmdCtx scaffold.WorkspaceContext) error {
	featuresPath := filepath.Join(scaffold.DefaultBundleDir(cmdCtx.GoRoot), "features.go")
	if scaffold.FileExists(featuresPath) {
		return nil
	}

	legacyPath := filepath.Join(scaffold.LegacyDefaultBundleDir(cmdCtx.GoRoot), "features.go")
	if scaffold.FileExists(legacyPath) {
		content, err := os.ReadFile(legacyPath)
		if err != nil {
			return fmt.Errorf("read cmd/%s/features.go: %w", scaffold.LegacyDefaultBundleName, err)
		}

		if err := os.WriteFile(featuresPath, content, 0o644); err != nil {
			return err
		}

		cmdCtx.Log.Printf("Copied features.go from cmd/%s/\n", scaffold.LegacyDefaultBundleName)

		return nil
	}

	imports, err := scaffold.DiscoverFeatureModuleImports(cmdCtx.GoRoot, cmdCtx.RepoModule, scaffold.DefaultFeaturePrefix)
	if err != nil {
		return err
	}

	if err := os.WriteFile(featuresPath, []byte(scaffold.RenderFeaturesGo(imports)), 0o644); err != nil {
		return err
	}

	if len(imports) > 0 {
		cmdCtx.Log.Printf(
			"Created cmd/%s/features.go with %d feature import(s) discovered under %s/\n",
			scaffold.DefaultBundleName,
			len(imports),
			opt.FeatureContainerFolder,
		)
	} else {
		cmdCtx.Log.Printf("Created empty cmd/%s/features.go\n", scaffold.DefaultBundleName)
	}

	return nil
}

func renameLegacyServiceBundle(cmdCtx scaffold.WorkspaceContext) error {
	legacy := scaffold.LegacyDefaultBundleDir(cmdCtx.GoRoot)
	current := scaffold.DefaultBundleDir(cmdCtx.GoRoot)

	if scaffold.FileExists(filepath.Join(legacy, "main.go")) {
		if scaffold.FileExists(filepath.Join(current, "main.go")) {
			if err := os.RemoveAll(legacy); err != nil {
				return fmt.Errorf("remove legacy cmd/%s: %w", scaffold.LegacyDefaultBundleName, err)
			}

			cmdCtx.Log.Printf("Removed legacy cmd/%s/\n", scaffold.LegacyDefaultBundleName)
		} else if err := os.Rename(legacy, current); err != nil {
			return fmt.Errorf("rename cmd/%s to cmd/%s: %w", scaffold.LegacyDefaultBundleName, scaffold.DefaultBundleName, err)
		} else {
			cmdCtx.Log.Printf("Renamed cmd/%s/ to cmd/%s/\n", scaffold.LegacyDefaultBundleName, scaffold.DefaultBundleName)
		}

		goModPath := filepath.Join(current, "go.mod")
		if content, err := os.ReadFile(goModPath); err == nil {
			oldMod := path.Join(opt.RepositoryGoModule, "cmd", scaffold.LegacyDefaultBundleName)
			newMod := path.Join(opt.RepositoryGoModule, "cmd", scaffold.DefaultBundleName)
			updated := strings.ReplaceAll(string(content), oldMod, newMod)

			if err := os.WriteFile(goModPath, []byte(updated), 0o644); err != nil {
				return fmt.Errorf("update cmd/%s/go.mod: %w", scaffold.DefaultBundleName, err)
			}
		}

		workDir := z.RunCmdParams{Dir: cmdCtx.GoRoot}
		if err := z.RunCmd(
			context.Background(),
			workDir,
			"go",
			"work",
			"use",
			"./cmd/"+scaffold.DefaultBundleName,
		); err != nil {
			return err
		}

		_ = z.RunCmd(
			context.Background(),
			workDir,
			"go",
			"work",
			"edit",
			"-dropuse",
			"./cmd/"+scaffold.LegacyDefaultBundleName,
		)
	}

	return migrateLegacyBundlesManifest(cmdCtx)
}

func migrateLegacyBundlesManifest(cmdCtx scaffold.WorkspaceContext) error {
	cfg, err := scaffold.LoadBundlesConfig(cmdCtx.GoRoot)
	if err != nil {
		return err
	}

	spec, ok := cfg.Bundles[scaffold.LegacyDefaultBundleName]
	if !ok {
		return nil
	}

	if _, exists := cfg.Bundles[scaffold.DefaultBundleName]; !exists {
		cfg.Bundles[scaffold.DefaultBundleName] = spec
	}

	delete(cfg.Bundles, scaffold.LegacyDefaultBundleName)

	if err := scaffold.SaveBundlesConfig(cmdCtx.GoRoot, cfg); err != nil {
		return err
	}

	cmdCtx.Log.Printf(
		"Renamed %q bundle to %q in %s\n",
		scaffold.LegacyDefaultBundleName,
		scaffold.DefaultBundleName,
		scaffold.BundlesManifestName,
	)

	return nil
}
