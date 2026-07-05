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
	Short: "Migrate a legacy workspace to the bundle + internal/runner layout",
	Long: `Upgrade workspaces created before executable bundles were introduced.

The command is idempotent: safe to run multiple times. It will:

  - add internal/runner/ (shared bootstrap) when missing or outdated
  - rewrite cmd/service/main.go to delegate to internal/runner
  - remove legacy cmd/service/middleware.go and healthz.go
  - create bundles.yaml when missing (default service bundle uses "*")
  - register internal/runner in go.work and refresh bundle features.go files

Run from the Go workspace root (repository root for backend workspaces, or backend/ in fullstack).

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
	HasRunnerModule  bool
	HasRunnerPackage bool
	HasBundlesYAML   bool
	UsesRunnerMain   bool
	LegacyMain       bool
	LegacyMiddleware bool
	LegacyHealthz    bool
}

func (s bundleLayoutStatus) NeedsMigration() bool {
	if s.LegacyMiddleware || s.LegacyHealthz || s.LegacyMain {
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
		HasRunnerModule:  fileExists(filepath.Join(goRoot, "internal", "runner", "go.mod")),
		HasRunnerPackage: fileExists(filepath.Join(goRoot, "internal", "runner", "runner.go")),
		HasBundlesYAML:   fileExists(bundlesManifestPath(goRoot)),
	}

	mainPath := filepath.Join(goRoot, "cmd", defaultBundleName, "main.go")
	if content, err := os.ReadFile(mainPath); err == nil {
		text := string(content)
		status.UsesRunnerMain = strings.Contains(text, "/internal/runner") &&
			strings.Contains(text, "runner.Execute")
		status.LegacyMain = strings.Contains(text, "genServerProvider") ||
			strings.Contains(text, "newRootCommand") ||
			(strings.Contains(text, "cobra.Command") && !status.UsesRunnerMain)
	}

	status.LegacyMiddleware = fileExists(filepath.Join(goRoot, "cmd", defaultBundleName, "middleware.go"))
	status.LegacyHealthz = fileExists(filepath.Join(goRoot, "cmd", defaultBundleName, "healthz.go"))

	return status
}

func runMigrateBundles(cmd *cobra.Command) error {
	cwd := rkit.GetCurrentDir()

	ok, err := isGoWorkspaceRoot(cwd)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("run this command in a go workspace root directory")
	}

	if f := cmd.Flag("repoModule"); f == nil || !f.Changed {
		detected, err := detectGoModule(cwd)
		if err != nil {
			return fmt.Errorf("could not auto-detect repository go module: %w", err)
		}

		opt.RepositoryGoModule = detected
	}

	status := detectBundleLayout(cwd)

	cmdCtx := workspaceCommandContext{
		cmd:        cmd,
		goRoot:     cwd,
		repoModule: opt.RepositoryGoModule,
	}

	if status.IsCurrent() {
		cmd.Println("Workspace already uses the bundle layout")

		if migrateOpt.DryRun {
			return nil
		}

		return syncAllBundleFeatures(cmdCtx)
	}

	appName, err := detectApplicationName(cwd)
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
		steps = append(steps, "copy internal/runner/ from scaffold")
	} else if status.LegacyMiddleware || status.LegacyHealthz || status.LegacyMain {
		steps = append(steps, "refresh internal/runner/ scaffold files")
	}

	if status.LegacyMain || !status.UsesRunnerMain {
		steps = append(steps, "rewrite cmd/service/main.go to use internal/runner")
	}

	if status.LegacyMiddleware {
		steps = append(steps, "remove cmd/service/middleware.go")
	}

	if status.LegacyHealthz {
		steps = append(steps, "remove cmd/service/healthz.go")
	}

	if !status.HasBundlesYAML {
		steps = append(steps, "create bundles.yaml with default service bundle")
	}

	if !status.HasRunnerModule {
		steps = append(steps, "initialize internal/runner go module and go work use")
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

	if status.LegacyMain || !status.UsesRunnerMain {
		if err := backupLegacyMain(cmdCtx); err != nil {
			return err
		}

		if err := renderServiceMain(filepath.Join(cmdCtx.goRoot, "cmd", defaultBundleName), templateInput); err != nil {
			return err
		}

		cmdCtx.cmd.Println("Updated cmd/service/main.go")
	}

	for _, rel := range []string{"middleware.go", "healthz.go"} {
		path := filepath.Join(cmdCtx.goRoot, "cmd", defaultBundleName, rel)
		if !fileExists(path) {
			continue
		}

		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove cmd/service/%s: %w", rel, err)
		}

		cmdCtx.cmd.Printf("Removed cmd/service/%s\n", rel)
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

	if err := tidyServiceModule(cmdCtx); err != nil {
		return err
	}

	return syncAllBundleFeatures(cmdCtx)
}

func copyRunnerScaffold(goRoot string, templateInput TemplateInput, overwrite bool) error {
	dest := filepath.Join(goRoot, "internal", "runner")

	return z.CopyDir(z.CopyDirParams{
		FS:             internal.Skeleton,
		SrcPathPrefix:  filepath.Join("skeleton", "backend", "internal", "runner"),
		DestPathPrefix: dest,
		TemplateInput:  templateInput,
		SkipExisting:   !overwrite,
	})
}

func renderServiceMain(bundleDir string, templateInput TemplateInput) error {
	srcPath := filepath.Join("skeleton", "backend", "cmd", "service", "main.gotmpl")
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
	mainPath := filepath.Join(cmdCtx.goRoot, "cmd", defaultBundleName, "main.go")
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

	if strings.Contains(string(content), "/internal/runner") {
		return nil
	}

	if err := os.WriteFile(backupPath, content, 0o644); err != nil {
		return fmt.Errorf("backup legacy main.go: %w", err)
	}

	cmdCtx.cmd.Println("Backed up legacy main.go to cmd/service/main.go.legacy")

	return nil
}

func seedBundlesManifest(goRoot string) error {
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
	runnerDir := filepath.Join(cmdCtx.goRoot, "internal", "runner")
	modulePath := path.Join(opt.RepositoryGoModule, "internal", "runner")
	p := z.RunCmdParams{Dir: runnerDir}

	if !fileExists(filepath.Join(runnerDir, "go.mod")) {
		z.RunCmd(context.Background(), p, "go", "mod", "init", modulePath)
		z.RunCmd(context.Background(), p, "go", "mod", "edit", "-go=1.25")
		cmdCtx.cmd.Println("Initialized internal/runner module")
	}

	z.RunCmd(context.Background(), p, "go", "mod", "tidy", "-e")
	z.RunCmd(context.Background(), p, "go", "fmt", "./...")

	workDir := z.RunCmdParams{Dir: cmdCtx.goRoot}
	z.RunCmd(context.Background(), workDir, "go", "work", "use", "./internal/runner")

	return nil
}

func tidyServiceModule(cmdCtx workspaceCommandContext) error {
	p := z.RunCmdParams{Dir: filepath.Join(cmdCtx.goRoot, "cmd", defaultBundleName)}
	z.RunCmd(context.Background(), p, "go", "mod", "tidy", "-e")
	z.RunCmd(context.Background(), p, "go", "fmt", "./...")

	return nil
}
