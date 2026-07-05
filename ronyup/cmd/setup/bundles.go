package setup

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultBundleName   = "service"
	bundlesManifestName = "bundles.yaml"
	wildcardService     = "*"
)

type BundlesConfig struct {
	Bundles map[string]BundleSpec `yaml:"bundles"`
}

type BundleSpec struct {
	Description string   `yaml:"description,omitempty"`
	Services    []string `yaml:"services"`
}

func bundlesManifestPath(goRoot string) string {
	return filepath.Join(goRoot, bundlesManifestName)
}

func loadBundlesConfig(goRoot string) (BundlesConfig, error) {
	data, err := os.ReadFile(bundlesManifestPath(goRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return BundlesConfig{
				Bundles: map[string]BundleSpec{
					defaultBundleName: {
						Description: "All-in-one dev binary (imports every feature)",
						Services:    []string{wildcardService},
					},
				},
			}, nil
		}

		return BundlesConfig{}, err
	}

	var cfg BundlesConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return BundlesConfig{}, fmt.Errorf("parse %s: %w", bundlesManifestName, err)
	}

	if cfg.Bundles == nil {
		cfg.Bundles = map[string]BundleSpec{}
	}

	return cfg, nil
}

func saveBundlesConfig(goRoot string, cfg BundlesConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(bundlesManifestPath(goRoot), data, 0o644)
}

func parseFeatureImports(featuresGoPath string) ([]string, error) {
	content, err := os.ReadFile(featuresGoPath)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, featuresGoPath, content, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	imports := make([]string, 0, len(file.Imports))
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if importPath != "" {
			imports = append(imports, importPath)
		}
	}

	slices.Sort(imports)

	return imports, nil
}

func featureImportPath(repoModule, featurePackagePath string) string {
	return path.Join(repoModule, featurePackagePath)
}

func importSuffix(importPath, repoModule string) string {
	repoModule = strings.TrimSuffix(repoModule, "/")
	if after, ok := strings.CutPrefix(importPath, repoModule+"/"); ok {
		return after
	}

	return importPath
}

func bundleIncludesService(bundle BundleSpec, servicePath string) bool {
	for _, svc := range bundle.Services {
		if svc == wildcardService || svc == servicePath {
			return true
		}
	}

	return false
}

func filterImportsForBundle(allImports []string, repoModule string, bundle BundleSpec) ([]string, error) {
	if len(bundle.Services) == 0 {
		return nil, fmt.Errorf("bundle must declare at least one service or %q", wildcardService)
	}

	if slices.Contains(bundle.Services, wildcardService) {
		return slices.Clone(allImports), nil
	}

	selected := make([]string, 0, len(bundle.Services))

	known := map[string]string{}
	for _, imp := range allImports {
		known[importSuffix(imp, repoModule)] = imp
	}

	for _, svc := range bundle.Services {
		imp, ok := known[svc]
		if !ok {
			return nil, fmt.Errorf("service %q is not imported in cmd/%s/features.go", svc, defaultBundleName)
		}

		selected = append(selected, imp)
	}

	slices.Sort(selected)

	return selected, nil
}

func renderFeaturesGo(imports []string) string {
	var b strings.Builder

	b.WriteString("package main\n\n")
	b.WriteString("/*\n\tFeatures MUST be imported here\n*/\n")

	if len(imports) == 0 {
		return b.String()
	}

	b.WriteString("\nimport (\n")

	for _, imp := range imports {
		fmt.Fprintf(&b, "\t_ %q\n", imp)
	}

	b.WriteString(")\n")

	return b.String()
}

func writeFeaturesGo(bundleDir string, imports []string) error {
	content := renderFeaturesGo(imports)

	return os.WriteFile(filepath.Join(bundleDir, "features.go"), []byte(content), 0o644)
}

func syncBundleFeatures(
	goRoot, repoModule, bundleName string,
	bundle BundleSpec,
	allImports []string,
) error {
	imports, err := filterImportsForBundle(allImports, repoModule, bundle)
	if err != nil {
		return err
	}

	bundleDir := filepath.Join(goRoot, "cmd", bundleName)

	return writeFeaturesGo(bundleDir, imports)
}

func syncAllBundleFeatures(cmdCtx workspaceCommandContext) error {
	cfg, err := loadBundlesConfig(cmdCtx.goRoot)
	if err != nil {
		return err
	}

	allImports, err := parseFeatureImports(filepath.Join(cmdCtx.goRoot, "cmd", defaultBundleName, "features.go"))
	if err != nil {
		return fmt.Errorf("read cmd/%s/features.go: %w", defaultBundleName, err)
	}

	for name, spec := range cfg.Bundles {
		if err := syncBundleFeatures(cmdCtx.goRoot, cmdCtx.repoModule, name, spec, allImports); err != nil {
			return fmt.Errorf("bundle %q: %w", name, err)
		}

		cmdCtx.cmd.Printf("Synced cmd/%s/features.go\n", name)
	}

	return nil
}

func syncBundlesForFeature(cmdCtx workspaceCommandContext, featurePackagePath string) error {
	cfg, err := loadBundlesConfig(cmdCtx.goRoot)
	if err != nil {
		return err
	}

	allImports, err := parseFeatureImports(filepath.Join(cmdCtx.goRoot, "cmd", defaultBundleName, "features.go"))
	if err != nil {
		return fmt.Errorf("read cmd/%s/features.go: %w", defaultBundleName, err)
	}

	for name, spec := range cfg.Bundles {
		if name == defaultBundleName {
			continue
		}

		hasWildcard := slices.Contains(spec.Services, wildcardService)
		if !hasWildcard && !bundleIncludesService(spec, featurePackagePath) {
			continue
		}

		if err := syncBundleFeatures(cmdCtx.goRoot, cmdCtx.repoModule, name, spec, allImports); err != nil {
			return fmt.Errorf("bundle %q: %w", name, err)
		}

		cmdCtx.cmd.Printf("Updated cmd/%s/features.go\n", name)
	}

	return nil
}

type workspaceCommandContext struct {
	cmd        commandPrinter
	goRoot     string
	repoModule string
}

type commandPrinter interface {
	Printf(format string, args ...any)
	Println(args ...any)
	PrintErrf(format string, args ...any)
}

func resolveFeaturePackagePath() string {
	groupFolder := ""
	if opt.GroupByTemplate {
		groupFolder = opt.Template
	}

	return path.Join(opt.FeatureContainerFolder, groupFolder, opt.FeatureDir)
}
