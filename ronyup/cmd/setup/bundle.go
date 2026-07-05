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

var bundleOpt = struct {
	Name        string
	Services    []string
	Description string
	Gen         bool
	Remove      bool
}{}

var CmdSetupBundle = &cobra.Command{
	Use:   "bundle",
	Short: "Create or refresh executable bundles that mix and match feature services",
	Long: `Bundles declare which feature modules are compiled into each cmd/<name>/ executable.

The default "service" bundle is the all-in-one dev binary. Additional bundles get their
own cmd/<name>/ module with a selective features.go import list.

Examples:
  ronyup setup bundle --name auth-api --services feature/auth,feature/session
  ronyup setup bundle --gen
  ronyup setup bundle --remove auth-api`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}

		return runBundle(cmd)
	},
}

func init() {
	flags := CmdSetupBundle.Flags()
	flags.StringVarP(&bundleOpt.Name, "name", "n", "", "bundle name (cmd/<name>/ directory)")
	flags.StringSliceVarP(
		&bundleOpt.Services,
		"services",
		"s",
		nil,
		"feature module paths (settings.ModuleName), or * for all",
	)
	flags.StringVar(&bundleOpt.Description, "description", "", "optional bundle description stored in bundles.yaml")
	flags.BoolVar(&bundleOpt.Gen, "gen", false, "regenerate features.go for every bundle from bundles.yaml")
	flags.BoolVar(&bundleOpt.Remove, "remove", false, "remove a bundle from bundles.yaml and delete cmd/<name>/")

	Cmd.AddCommand(CmdSetupBundle)
}

func runBundle(cmd *cobra.Command) error {
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

	cmdCtx := workspaceCommandContext{
		cmd:        cmd,
		goRoot:     cwd,
		repoModule: opt.RepositoryGoModule,
	}

	if bundleOpt.Gen {
		return syncAllBundleFeatures(cmdCtx)
	}

	if bundleOpt.Remove {
		if bundleOpt.Name == "" {
			return fmt.Errorf("--name is required with --remove")
		}

		return removeBundle(cmdCtx)
	}

	if bundleOpt.Name == "" {
		return fmt.Errorf("--name is required")
	}

	if bundleOpt.Name == defaultBundleName {
		return fmt.Errorf("bundle %q is managed by setup workspace; use --gen to refresh it", defaultBundleName)
	}

	if len(bundleOpt.Services) == 0 {
		return fmt.Errorf("--services is required when creating a bundle")
	}

	return createBundle(cmdCtx)
}

func createBundle(cmdCtx workspaceCommandContext) error {
	cfg, err := loadBundlesConfig(cmdCtx.goRoot)
	if err != nil {
		return err
	}

	bundleDir := filepath.Join(cmdCtx.goRoot, "cmd", bundleOpt.Name)
	if z.IsEmptyDir(bundleDir) {
		if err := os.MkdirAll(bundleDir, 0o755); err != nil {
			return err
		}
	} else if !opt.Force {
		return fmt.Errorf("%s already exists, use -f to overwrite", filepath.Join("cmd", bundleOpt.Name))
	} else {
		if err := os.RemoveAll(bundleDir); err != nil {
			return err
		}

		if err := os.MkdirAll(bundleDir, 0o755); err != nil {
			return err
		}
	}

	cfg.Bundles[bundleOpt.Name] = BundleSpec{
		Description: bundleOpt.Description,
		Services:    append([]string(nil), bundleOpt.Services...),
	}

	if err := saveBundlesConfig(cmdCtx.goRoot, cfg); err != nil {
		return err
	}

	appName, err := detectApplicationName(cmdCtx.goRoot)
	if err != nil {
		return err
	}

	templateInput := TemplateInput{
		ApplicationName: appName,
		RepositoryPath:  strings.TrimSuffix(opt.RepositoryGoModule, "/"),
		PackageName:     appName,
		RonyKitPath:     "github.com/clubpay/ronykit",
		BundleName:      bundleOpt.Name,
	}

	if err := renderBundleMain(bundleDir, templateInput); err != nil {
		return err
	}

	allImports, err := parseFeatureImports(filepath.Join(cmdCtx.goRoot, "cmd", defaultBundleName, "features.go"))
	if err != nil {
		return fmt.Errorf("read cmd/%s/features.go: %w", defaultBundleName, err)
	}

	spec := cfg.Bundles[bundleOpt.Name]
	if err := syncBundleFeatures(cmdCtx.goRoot, cmdCtx.repoModule, bundleOpt.Name, spec, allImports); err != nil {
		return err
	}

	modulePath := path.Join(opt.RepositoryGoModule, "cmd", bundleOpt.Name)
	p := z.RunCmdParams{Dir: bundleDir}
	z.RunCmd(context.Background(), p, "go", "mod", "init", modulePath)
	z.RunCmd(context.Background(), p, "go", "mod", "edit", "-go=1.25")
	z.RunCmd(context.Background(), p, "go", "mod", "tidy", "-e")
	z.RunCmd(context.Background(), p, "go", "fmt", "./...")
	z.RunCmd(context.Background(), p, "go", "work", "use", ".")

	cmdCtx.cmd.Printf("Bundle %q created at cmd/%s/\n", bundleOpt.Name, bundleOpt.Name)

	return nil
}

func removeBundle(cmdCtx workspaceCommandContext) error {
	if bundleOpt.Name == defaultBundleName {
		return fmt.Errorf("cannot remove the default %q bundle", defaultBundleName)
	}

	cfg, err := loadBundlesConfig(cmdCtx.goRoot)
	if err != nil {
		return err
	}

	if _, ok := cfg.Bundles[bundleOpt.Name]; !ok {
		return fmt.Errorf("bundle %q not found in %s", bundleOpt.Name, bundlesManifestName)
	}

	delete(cfg.Bundles, bundleOpt.Name)

	if err := saveBundlesConfig(cmdCtx.goRoot, cfg); err != nil {
		return err
	}

	bundleDir := filepath.Join(cmdCtx.goRoot, "cmd", bundleOpt.Name)
	if err := os.RemoveAll(bundleDir); err != nil && !os.IsNotExist(err) {
		return err
	}

	p := z.RunCmdParams{Dir: cmdCtx.goRoot}
	z.RunCmd(context.Background(), p, "go", "work", "edit", "-dropuse", "./cmd/"+bundleOpt.Name)

	cmdCtx.cmd.Printf("Removed bundle %q\n", bundleOpt.Name)

	return nil
}

func renderBundleMain(bundleDir string, templateInput TemplateInput) error {
	srcPath := filepath.Join("skeleton", "backend", "cmd", "bundle", "main.gotmpl")
	destPath := filepath.Join(bundleDir, "main.go")

	return z.CopyFile(z.CopyFileParams{
		FS:             internal.Skeleton,
		SrcPath:        srcPath,
		DestPath:       destPath,
		TemplateSuffix: "tmpl",
		TemplateInput:  templateInput,
	})
}

func detectApplicationName(goRoot string) (string, error) {
	if opt.ApplicationName != "" {
		return opt.ApplicationName, nil
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
