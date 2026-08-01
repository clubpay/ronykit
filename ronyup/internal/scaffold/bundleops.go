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

// BundleRequest configures CreateBundle and RemoveBundle.
type BundleRequest struct {
	// StartDir is where Go workspace resolution begins.
	StartDir string
	// Module overrides auto-detection from go.work when non-empty.
	Module string
	// AppName overrides the application name used in templates when non-empty.
	AppName string
	// Name is the bundle name (cmd/<name>/ directory).
	Name string
	// Services lists feature module paths (settings.ModuleName), or * for all.
	Services []string
	// Description is optional text stored in bundles.yaml.
	Description string
	// Force replaces an existing non-empty cmd/<name>/ directory.
	Force bool
}

// RegenerateBundlesRequest configures RegenerateBundleFeatures.
type RegenerateBundlesRequest struct {
	// StartDir is where Go workspace resolution begins.
	StartDir string
	// Module overrides auto-detection from go.work when non-empty.
	Module string
}

func resolveBundleContext(reqStartDir, reqModule string, log Logger) (WorkspaceContext, error) {
	goRoot, err := resolveGoWorkspace(reqStartDir)
	if err != nil {
		return WorkspaceContext{}, err
	}

	module := reqModule
	if module == "" {
		module, err = detectGoModule(goRoot)
		if err != nil {
			return WorkspaceContext{}, fmt.Errorf("could not auto-detect repository go module: %w", err)
		}
	}

	log.Printf("Go workspace: %s\n", goRoot)

	return WorkspaceContext{
		Log:        log,
		GoRoot:     goRoot,
		RepoModule: module,
	}, nil
}

// RegenerateBundleFeatures rewrites features.go for every bundle declared in
// bundles.yaml (the `--gen` CLI mode).
func RegenerateBundleFeatures(ctx context.Context, req RegenerateBundlesRequest, log Logger) error {
	if log == nil {
		log = DiscardLogger{}
	}

	cmdCtx, err := resolveBundleContext(req.StartDir, req.Module, log)
	if err != nil {
		return err
	}

	return SyncAllBundleFeatures(cmdCtx)
}

// CreateBundle declares a new executable bundle in bundles.yaml and generates
// cmd/<name>/ with a selective features.go import list.
func CreateBundle(ctx context.Context, req BundleRequest, log Logger) error {
	if log == nil {
		log = DiscardLogger{}
	}

	if req.Name == "" {
		return fmt.Errorf("bundle name is required")
	}

	if req.Name == DefaultBundleName {
		return fmt.Errorf(
			"bundle %q is managed by setup workspace; regenerate features.go instead",
			DefaultBundleName,
		)
	}

	if len(req.Services) == 0 {
		return fmt.Errorf("services are required when creating a bundle (or %q for all)", WildcardService)
	}

	cmdCtx, err := resolveBundleContext(req.StartDir, req.Module, log)
	if err != nil {
		return err
	}

	cfg, err := LoadBundlesConfig(cmdCtx.GoRoot)
	if err != nil {
		return err
	}

	bundleDir := filepath.Join(cmdCtx.GoRoot, "cmd", req.Name)
	if z.IsEmptyDir(bundleDir) {
		if err := os.MkdirAll(bundleDir, 0o755); err != nil {
			return err
		}
	} else if !req.Force {
		return fmt.Errorf("%s already exists, use force to overwrite", filepath.Join("cmd", req.Name))
	} else {
		if err := os.RemoveAll(bundleDir); err != nil {
			return err
		}

		if err := os.MkdirAll(bundleDir, 0o755); err != nil {
			return err
		}
	}

	cfg.Bundles[req.Name] = BundleSpec{
		Description: req.Description,
		Services:    append([]string(nil), req.Services...),
	}

	if err := SaveBundlesConfig(cmdCtx.GoRoot, cfg); err != nil {
		return err
	}

	appName, err := detectApplicationName(cmdCtx.GoRoot, req.AppName)
	if err != nil {
		return err
	}

	templateInput := TemplateInput{
		ApplicationName: appName,
		RepositoryPath:  strings.TrimSuffix(cmdCtx.RepoModule, "/"),
		PackageName:     appName,
		RonyKitPath:     RonyKitModulePath,
		BundleName:      req.Name,
	}

	if err := renderBundleMain(bundleDir, templateInput); err != nil {
		return err
	}

	allImports, err := LoadDefaultBundleFeatureImports(cmdCtx)
	if err != nil {
		return err
	}

	spec := cfg.Bundles[req.Name]
	if err := SyncBundleFeatures(cmdCtx.GoRoot, cmdCtx.RepoModule, req.Name, spec, allImports); err != nil {
		return err
	}

	modulePath := path.Join(cmdCtx.RepoModule, "cmd", req.Name)

	p := z.RunCmdParams{Dir: bundleDir}
	if err := z.RunCmd(ctx, p, "go", "mod", "init", modulePath); err != nil {
		return err
	}

	if err := z.RunCmd(ctx, p, "go", "mod", "edit", "-go=1.25"); err != nil {
		return err
	}

	if err := z.RunCmd(ctx, p, "go", "mod", "tidy", "-e"); err != nil {
		return err
	}

	if err := z.RunCmd(ctx, p, "go", "fmt", "./..."); err != nil {
		return err
	}

	if err := z.RunCmd(ctx, p, "go", "work", "use", "."); err != nil {
		return err
	}

	log.Printf("Bundle %q created at cmd/%s/\n", req.Name, req.Name)

	return nil
}

// RemoveBundle deletes a bundle from bundles.yaml and removes cmd/<name>/.
func RemoveBundle(ctx context.Context, req BundleRequest, log Logger) error {
	if log == nil {
		log = DiscardLogger{}
	}

	if req.Name == "" {
		return fmt.Errorf("bundle name is required")
	}

	if req.Name == DefaultBundleName {
		return fmt.Errorf("cannot remove the default %q bundle", DefaultBundleName)
	}

	cmdCtx, err := resolveBundleContext(req.StartDir, req.Module, log)
	if err != nil {
		return err
	}

	cfg, err := LoadBundlesConfig(cmdCtx.GoRoot)
	if err != nil {
		return err
	}

	if _, ok := cfg.Bundles[req.Name]; !ok {
		return fmt.Errorf("bundle %q not found in %s", req.Name, BundlesManifestName)
	}

	delete(cfg.Bundles, req.Name)

	if err := SaveBundlesConfig(cmdCtx.GoRoot, cfg); err != nil {
		return err
	}

	bundleDir := filepath.Join(cmdCtx.GoRoot, "cmd", req.Name)
	if err := os.RemoveAll(bundleDir); err != nil && !os.IsNotExist(err) {
		return err
	}

	p := z.RunCmdParams{Dir: cmdCtx.GoRoot}
	// Best-effort: the use directive may already be absent.
	_ = z.RunCmd(ctx, p, "go", "work", "edit", "-dropuse", "./cmd/"+req.Name)

	log.Printf("Removed bundle %q\n", req.Name)

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
