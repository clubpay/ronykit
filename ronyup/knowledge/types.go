package knowledge

// Base holds all parsed knowledge loaded from embedded Markdown files.
type Base struct {
	ServerInstructions string
	Tools              map[string]ToolDoc
	Packages           []PackageDoc
	ArchitectureHints  []ArchitectureHint
	Characteristics    []CharacteristicDoc
	PlanNextSteps      []string
	FilePurposes       map[string]string
	Prompts            []PromptDoc
}

// ToolDoc describes an MCP tool loaded from tools/*.md.
type ToolDoc struct {
	Name             string `yaml:"name"`
	Description      string
	ExtendedGuidance string
}

// PackageDoc describes a recommended x/ toolkit package loaded from packages/*.md.
type PackageDoc struct {
	ImportPath  string `yaml:"import_path"`
	ShortName   string `yaml:"short_name"`
	Description string
	UsageHint   string
}

// ArchitectureHint is a single best-practice hint loaded from architecture/*.md.
type ArchitectureHint struct {
	Slug string
	Text string
}

// CharacteristicDoc describes a service characteristic loaded from characteristics/*.md.
type CharacteristicDoc struct {
	Keywords       []string `yaml:"keywords"`
	AppliesToFiles []string `yaml:"applies_to_files"`
	ServiceHint    string
	FileHint       string
}

// PromptDoc describes an MCP prompt template loaded from prompts/*.md.
type PromptDoc struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Arguments   []PromptArgument `yaml:"arguments"`
	Template    string
}

// PromptArgument is a single argument for a prompt template.
type PromptArgument struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
}
