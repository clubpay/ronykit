package mcp

// -- Server metadata --------------------------------------------------------

const serverTitle = "RonyUP MCP Server"

// -- Tool names -------------------------------------------------------------

const (
	toolListTemplates    = "list_templates"
	toolReadTemplate     = "read_template"
	toolCreateWorkspace  = "create_workspace"
	toolCreateFeature    = "create_feature"
	toolPlanService      = "plan_service"
	toolImplementService = "implement_service"
)

// -- Result messages (format templates) -------------------------------------

const (
	msgFoundTemplates   = "Found %d scaffold files/templates."
	msgLoadedTemplate   = "Loaded template %s (%d bytes)."
	msgWorkspaceCreated = "Workspace created successfully."
	msgFeatureCreated   = "Feature created successfully."
	msgPlannedFiles     = "Planned %d files for service feature %s."
	msgImplemented      = "Implemented service starter code in %d file(s)."
)

// -- Error messages (format templates) --------------------------------------

const (
	errTemplatePathRequired = "template path is required"
	errInvalidTemplatePath  = "invalid template path: %q"
	errPathRequired         = "path is required"
	errPathTraversal        = "path must be relative and without traversal: %q"
	errFeatureNameRequired  = "feature_name is required"
	errFeatureNameInvalid   = "feature_name must be a valid Go identifier: %q"
	errWorkspaceNotDir      = "workspace_dir must be a directory: %q"
	errWorkspaceNoGoWork    = "workspace_dir must contain go.work: %w"
	errWorkspaceSetupFailed = "workspace setup failed: %w, stderr: %s"
	errFeatureSetupFailed   = "feature setup failed: %w, stderr: %s"
	errFeaturePathMissing   = "feature path does not exist: %s " +
		"(set create_if_missing=true to scaffold first)"
	errCreateFeatureFailed  = "create feature failed: %w, stderr: %s"
	errInvalidFeatureDir    = "invalid feature_dir: %w"
	errContractNameRequired = "contract name is required"
	errDuplicateContract    = "duplicate contract name: %s"
	errUnsupportedMethod    = "unsupported http_method for %s: %s"
	errGofmtFailed          = "gofmt failed: %w, stderr: %s"
)

// -- Domain constants -------------------------------------------------------

const (
	serviceTemplateName      = "service"
	ronyKitModulePath        = "github.com/clubpay/ronykit"
	defaultServiceIdentifier = "Service"
	defaultServiceGroup      = "service"
)

// -- Recommended packages output type ---------------------------------------

type toolkitPackage struct {
	ImportPath  string `json:"import_path"`
	ShortName   string `json:"short_name"`
	Description string `json:"description"`
	UsageHint   string `json:"usage_hint"`
}
