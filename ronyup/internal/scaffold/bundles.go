package scaffold

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultBundleName is the primary executable bundle (cmd/all-in-one).
	DefaultBundleName = "all-in-one"
	// LegacyDefaultBundleName is the pre-migration default bundle name.
	LegacyDefaultBundleName = "service"
	// DefaultFeaturePrefix is the default parent directory for feature modules.
	DefaultFeaturePrefix = "feature"
	// BundlesManifestName is the bundles config filename under the Go workspace root.
	BundlesManifestName = "bundles.yaml"
	// WildcardService selects every feature module in a bundle.
	WildcardService = "*"
)

const (
	bundlesManifestName = BundlesManifestName
	wildcardService     = WildcardService
)

const (
	defaultBundleName       = DefaultBundleName
	legacyDefaultBundleName = LegacyDefaultBundleName
)

func defaultBundleDir(goRoot string) string {
	return filepath.Join(goRoot, "cmd", defaultBundleName)
}

func legacyDefaultBundleDir(goRoot string) string {
	return filepath.Join(goRoot, "cmd", legacyDefaultBundleName)
}

// DefaultBundleDir returns cmd/<DefaultBundleName> under goRoot.
func DefaultBundleDir(goRoot string) string { return defaultBundleDir(goRoot) }

// LegacyDefaultBundleDir returns cmd/<LegacyDefaultBundleName> under goRoot.
func LegacyDefaultBundleDir(goRoot string) string { return legacyDefaultBundleDir(goRoot) }

// LoadBundlesConfig reads bundles.yaml (or returns the default config).
func LoadBundlesConfig(goRoot string) (BundlesConfig, error) { return loadBundlesConfig(goRoot) }

// SaveBundlesConfig writes bundles.yaml.
func SaveBundlesConfig(goRoot string, cfg BundlesConfig) error { return saveBundlesConfig(goRoot, cfg) }

// SyncBundleFeatures writes features.go for one bundle.
func SyncBundleFeatures(
	goRoot, repoModule, bundleName string, bundle BundleSpec, allImports []string,
) error {
	return syncBundleFeatures(goRoot, repoModule, bundleName, bundle, allImports)
}

type BundlesConfig struct {
	Bundles map[string]BundleSpec `yaml:"bundles"`
}

type BundleSpec struct {
	Description string   `yaml:"description,omitempty"`
	Services    []string `yaml:"services"`
}

// BundlesManifestPath returns goRoot/bundles.yaml.
func BundlesManifestPath(goRoot string) string { return bundlesManifestPath(goRoot) }

func bundlesManifestPath(goRoot string) string {
	return filepath.Join(goRoot, bundlesManifestName)
}

// RenderFeaturesGo renders the blank-import features.go contents.
func RenderFeaturesGo(imports []string) string { return renderFeaturesGo(imports) }

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

// WorkspaceContext carries Go workspace location and a progress logger for
// bundle/migrate operations.
type WorkspaceContext struct {
	Log        Logger
	GoRoot     string
	RepoModule string
}

func (c WorkspaceContext) log() Logger {
	if c.Log == nil {
		return DiscardLogger{}
	}

	return c.Log
}

// SyncAllBundleFeatures regenerates features.go for every bundle in bundles.yaml.
func SyncAllBundleFeatures(cmdCtx WorkspaceContext) error {
	return syncAllBundleFeatures(cmdCtx)
}

func syncAllBundleFeatures(cmdCtx WorkspaceContext) error {
	cfg, err := loadBundlesConfig(cmdCtx.GoRoot)
	if err != nil {
		return err
	}

	allImports, err := loadDefaultBundleFeatureImports(cmdCtx)
	if err != nil {
		return err
	}

	for name, spec := range cfg.Bundles {
		if err := syncBundleFeatures(cmdCtx.GoRoot, cmdCtx.RepoModule, name, spec, allImports); err != nil {
			return fmt.Errorf("bundle %q: %w", name, err)
		}

		cmdCtx.log().Printf("Synced cmd/%s/features.go\n", name)
	}

	return nil
}

// SyncBundlesForFeature updates non-default bundles that include the feature.
func SyncBundlesForFeature(cmdCtx WorkspaceContext, featurePackagePath string) error {
	return syncBundlesForFeature(cmdCtx, featurePackagePath)
}

func syncBundlesForFeature(cmdCtx WorkspaceContext, featurePackagePath string) error {
	cfg, err := loadBundlesConfig(cmdCtx.GoRoot)
	if err != nil {
		return err
	}

	allImports, err := loadDefaultBundleFeatureImports(cmdCtx)
	if err != nil {
		return err
	}

	for name, spec := range cfg.Bundles {
		if name == defaultBundleName {
			continue
		}

		hasWildcard := slices.Contains(spec.Services, wildcardService)
		if !hasWildcard && !bundleIncludesService(spec, featurePackagePath) {
			continue
		}

		if err := syncBundleFeatures(cmdCtx.GoRoot, cmdCtx.RepoModule, name, spec, allImports); err != nil {
			return fmt.Errorf("bundle %q: %w", name, err)
		}

		cmdCtx.log().Printf("Updated cmd/%s/features.go\n", name)
	}

	return nil
}

// FeaturePackagePath returns the workspace-relative feature module path.
func FeaturePackagePath(prefix, template, featureDir string, groupByTemplate bool) string {
	if prefix == "" {
		prefix = DefaultFeaturePrefix
	}

	groupFolder := ""
	if groupByTemplate {
		groupFolder = template
	}

	return path.Join(prefix, groupFolder, featureDir)
}

// LoadDefaultBundleFeatureImports reads feature imports from the default bundle.
func LoadDefaultBundleFeatureImports(cmdCtx WorkspaceContext) ([]string, error) {
	return loadDefaultBundleFeatureImports(cmdCtx)
}

func loadDefaultBundleFeatureImports(cmdCtx WorkspaceContext) ([]string, error) {
	for _, bundleName := range []string{defaultBundleName, legacyDefaultBundleName} {
		featuresPath := filepath.Join(cmdCtx.GoRoot, "cmd", bundleName, "features.go")

		imports, err := parseFeatureImports(featuresPath)
		if err == nil {
			return imports, nil
		}

		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read cmd/%s/features.go: %w", bundleName, err)
		}
	}

	return nil, fmt.Errorf(
		"read cmd/%s/features.go: no such file or directory (run `ronyup setup migrate bundles`)",
		defaultBundleName,
	)
}

// DiscoverFeatureModuleImports finds feature modules under featurePrefix.
func DiscoverFeatureModuleImports(goRoot, repoModule, featurePrefix string) ([]string, error) {
	return discoverFeatureModuleImports(goRoot, repoModule, featurePrefix)
}

func discoverFeatureModuleImports(goRoot, repoModule, featurePrefix string) ([]string, error) {
	if featurePrefix == "" {
		featurePrefix = DefaultFeaturePrefix
	}

	featureRoot := filepath.Join(goRoot, featurePrefix)
	if !fileExists(featureRoot) {
		return nil, nil
	}

	imports := make([]string, 0)

	err := filepath.WalkDir(featureRoot, func(walkPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if !entry.IsDir() || walkPath == featureRoot {
			return nil
		}

		if !fileExists(filepath.Join(walkPath, "go.mod")) {
			return nil
		}

		rel, err := filepath.Rel(goRoot, walkPath)
		if err != nil {
			return err
		}

		imports = append(imports, path.Join(repoModule, filepath.ToSlash(rel)))

		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}

	slices.Sort(imports)

	return imports, nil
}
