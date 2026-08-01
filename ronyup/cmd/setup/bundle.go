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

The default "all-in-one" bundle is the all-in-one dev binary. Additional bundles get their
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
	flags.StringVar(
		&bundleOpt.Description,
		"description",
		"",
		"optional bundle description stored in bundles.yaml",
	)
	flags.BoolVar(&bundleOpt.Gen, "gen", false, "regenerate features.go for every bundle from bundles.yaml")
	flags.BoolVar(
		&bundleOpt.Remove,
		"remove",
		false,
		"remove a bundle from bundles.yaml and delete cmd/<name>/",
	)

	Cmd.AddCommand(CmdSetupBundle)
}

func runBundle(cmd *cobra.Command) error {
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

	cmdCtx := scaffold.WorkspaceContext{
		Log:        cmd,
		GoRoot:     goRoot,
		RepoModule: opt.RepositoryGoModule,
	}

	cmd.Printf("Go workspace: %s\n", goRoot)

	if bundleOpt.Gen {
		return scaffold.SyncAllBundleFeatures(cmdCtx)
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

	if bundleOpt.Name == scaffold.DefaultBundleName {
		return fmt.Errorf(
			"bundle %q is managed by setup workspace; use --gen to refresh it",
			scaffold.DefaultBundleName,
		)
	}

	if len(bundleOpt.Services) == 0 {
		return fmt.Errorf("--services is required when creating a bundle")
	}

	return createBundle(cmdCtx)
}

func createBundle(cmdCtx scaffold.WorkspaceContext) error {
	cfg, err := scaffold.LoadBundlesConfig(cmdCtx.GoRoot)
	if err != nil {
		return err
	}

	bundleDir := filepath.Join(cmdCtx.GoRoot, "cmd", bundleOpt.Name)
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

	cfg.Bundles[bundleOpt.Name] = scaffold.BundleSpec{
		Description: bundleOpt.Description,
		Services:    append([]string(nil), bundleOpt.Services...),
	}

	if err := scaffold.SaveBundlesConfig(cmdCtx.GoRoot, cfg); err != nil {
		return err
	}

	appName, err := detectApplicationName(cmdCtx.GoRoot)
	if err != nil {
		return err
	}

	templateInput := scaffold.TemplateInput{
		ApplicationName: appName,
		RepositoryPath:  strings.TrimSuffix(opt.RepositoryGoModule, "/"),
		PackageName:     appName,
		RonyKitPath:     scaffold.RonyKitModulePath,
		BundleName:      bundleOpt.Name,
	}

	if err := renderBundleMain(bundleDir, templateInput); err != nil {
		return err
	}

	allImports, err := scaffold.LoadDefaultBundleFeatureImports(cmdCtx)
	if err != nil {
		return err
	}

	spec := cfg.Bundles[bundleOpt.Name]
	if err := scaffold.SyncBundleFeatures(
		cmdCtx.GoRoot,
		cmdCtx.RepoModule,
		bundleOpt.Name,
		spec,
		allImports,
	); err != nil {
		return err
	}

	modulePath := path.Join(opt.RepositoryGoModule, "cmd", bundleOpt.Name)

	p := z.RunCmdParams{Dir: bundleDir}
	if err := z.RunCmd(context.Background(), p, "go", "mod", "init", modulePath); err != nil {
		return err
	}

	if err := z.RunCmd(context.Background(), p, "go", "mod", "edit", "-go=1.25"); err != nil {
		return err
	}

	if err := z.RunCmd(context.Background(), p, "go", "mod", "tidy", "-e"); err != nil {
		return err
	}

	if err := z.RunCmd(context.Background(), p, "go", "fmt", "./..."); err != nil {
		return err
	}

	if err := z.RunCmd(context.Background(), p, "go", "work", "use", "."); err != nil {
		return err
	}

	cmdCtx.Log.Printf("Bundle %q created at cmd/%s/\n", bundleOpt.Name, bundleOpt.Name)

	return nil
}

func removeBundle(cmdCtx scaffold.WorkspaceContext) error {
	if bundleOpt.Name == scaffold.DefaultBundleName {
		return fmt.Errorf("cannot remove the default %q bundle", scaffold.DefaultBundleName)
	}

	cfg, err := scaffold.LoadBundlesConfig(cmdCtx.GoRoot)
	if err != nil {
		return err
	}

	if _, ok := cfg.Bundles[bundleOpt.Name]; !ok {
		return fmt.Errorf("bundle %q not found in %s", bundleOpt.Name, scaffold.BundlesManifestName)
	}

	delete(cfg.Bundles, bundleOpt.Name)

	if err := scaffold.SaveBundlesConfig(cmdCtx.GoRoot, cfg); err != nil {
		return err
	}

	bundleDir := filepath.Join(cmdCtx.GoRoot, "cmd", bundleOpt.Name)
	if err := os.RemoveAll(bundleDir); err != nil && !os.IsNotExist(err) {
		return err
	}

	p := z.RunCmdParams{Dir: cmdCtx.GoRoot}
	// Best-effort: the use directive may already be absent.
	_ = z.RunCmd(context.Background(), p, "go", "work", "edit", "-dropuse", "./cmd/"+bundleOpt.Name)

	cmdCtx.Log.Printf("Removed bundle %q\n", bundleOpt.Name)

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

	module, err := scaffold.DetectGoModule(goRoot)
	if err != nil {
		return "", err
	}

	base := path.Base(module)
	if base == "" || base == "." {
		base = path.Base(goRoot)
	}

	return base, nil
}
