package scaffold

import "context"

// WorkspaceRequest configures SetupWorkspace.
type WorkspaceRequest struct {
	// Path is the destination directory for the workspace (repo root).
	Path string
	// Module is the Go module path (e.g. github.com/acme/app). Defaults when empty.
	Module string
	// AppName is the application name used in templates.
	AppName string
	// Kind is backend | fullstack | frontend.
	Kind string
	// Skills are skill IDs or tokens (default|all|none). Empty means kind defaults.
	Skills []string
	// Force removes an existing non-empty destination before scaffolding.
	Force bool
	// Custom is reserved for future template key/value overrides.
	Custom map[string]string
}

// FeatureRequest configures SetupFeature.
type FeatureRequest struct {
	// StartDir is where Go workspace resolution begins (CWD for CLI, workspacePath for MCP).
	StartDir string
	// Module overrides auto-detection from go.work when non-empty.
	Module string
	// FeatureDir is the feature directory name under FeaturePrefix.
	FeatureDir string
	// FeatureName is the Go package / feature name.
	FeatureName string
	// Template is service | job | gateway.
	Template string
	// FeaturePrefix is the parent directory for feature modules (default "feature").
	FeaturePrefix string
	// GroupByTemplate places the module under {prefix}/{template}/{name}/.
	GroupByTemplate bool
	// Force replaces a non-empty feature directory.
	Force bool
}

// FeatureTemplates are the supported feature skeleton names.
var FeatureTemplates = []string{"service", "job", "gateway"}

// WorkspaceKinds are the supported workspace layouts.
var WorkspaceKinds = []string{KindBackend, KindFullstack, KindFrontend}

// SetupWorkspace scaffolds a new workspace at req.Path.
func SetupWorkspace(ctx context.Context, req WorkspaceRequest, log Logger) error {
	if log == nil {
		log = DiscardLogger{}
	}

	return setupWorkspace(ctx, req, log)
}

// SetupFeature scaffolds a feature into an existing Go workspace.
func SetupFeature(ctx context.Context, req FeatureRequest, log Logger) error {
	if log == nil {
		log = DiscardLogger{}
	}

	return setupFeature(ctx, req, log)
}
