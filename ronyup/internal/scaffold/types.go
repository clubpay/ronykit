package scaffold

// TemplateInput is passed into skeleton template rendering.
type TemplateInput struct {
	ApplicationName string
	RepositoryPath  string
	// PackagePath is the folder that module will reside inside the Repository root folder
	PackagePath string
	// PackageName is the name of the package to be used for some internal variables
	PackageName string
	// RonyKitPath is the address of the RonyKIT modules
	RonyKitPath string
	// BundleName is the executable bundle name for cmd/<name>/ entrypoints.
	BundleName string
	// Kind is the workspace layout; templates use it to render layout-specific guidance.
	Kind string
	// Skills lists the agent skills pre-installed into .agents/skills so
	// templates (e.g. AGENTS.md) can reference them.
	Skills []SkillInfo
}

// RonyKitModulePath is the default import path for RonyKIT modules in scaffolds.
const RonyKitModulePath = "github.com/clubpay/ronykit"
