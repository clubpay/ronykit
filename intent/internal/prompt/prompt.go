package prompt

import (
	"embed"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var templates = mustParseTemplates()

func mustParseTemplates() *template.Template {
	tmpl := template.New("prompt").Funcs(template.FuncMap{
		"markdownCell": markdownTableCell,
		"join":         strings.Join,
		"add": func(a, b int) int {
			return a + b
		},
	})

	tmpl = template.Must(tmpl.ParseFS(templateFS, "templates/*.tmpl"))

	return tmpl
}

func render(name string, data any) string {
	var b strings.Builder
	if err := templates.ExecuteTemplate(&b, name, data); err != nil {
		panic("prompt: render " + name + ": " + err.Error())
	}

	return b.String()
}

func markdownTableCell(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", " ")

	return strings.TrimSpace(s)
}

// SkillCatalogTitle is the heading for templates/skill_catalog.tmpl.
const SkillCatalogTitle = "Skill Catalog"

// SkillRow is one entry in templates/skill_catalog.tmpl.
type SkillRow struct {
	Name        string   // templates/skill_catalog.tmpl: {{$skill.Name}}
	Description string   // templates/skill_catalog.tmpl: {{$skill.Description}}
	Triggers    []string // templates/skill_catalog.tmpl: {{join $skill.Triggers ", "}}
	Examples    []string // templates/skill_catalog.tmpl: {{range $skill.Examples}}
}

// SkillCatalogData is input for templates/skill_catalog.tmpl.
type SkillCatalogData struct {
	Title             string     // templates/skill_catalog.tmpl: {{.Title}}
	ActivateSkillTool string     // templates/skill_catalog.tmpl: {{.ActivateSkillTool}}
	Skills            []SkillRow // templates/skill_catalog.tmpl: {{range .Skills}}
}

// SkillCatalog renders the advertised skill index as markdown.
//
// Used by intent.FormatSkillCatalog → runtime.skillCatalogMessage.
func SkillCatalog(data SkillCatalogData) string {
	return strings.TrimRight(render("skill_catalog.tmpl", data), "\n")
}

// ActivateSkillToolDescriptionData is input for templates/activate_skill_tool.tmpl.
type ActivateSkillToolDescriptionData struct {
	CatalogTitle string // templates/activate_skill_tool.tmpl: {{printf "%q" .CatalogTitle}}
}

// ActivateSkillToolDescription renders the activate_skill tool description.
//
// Used by intent.activateSkillDefinition → ToolDefinition.Description.
func ActivateSkillToolDescription(data ActivateSkillToolDescriptionData) string {
	return strings.TrimSpace(render("activate_skill_tool.tmpl", data))
}

// ActivateSkillNameDescriptionData is input for templates/activate_skill_name.tmpl.
type ActivateSkillNameDescriptionData struct {
	CatalogTitle string // templates/activate_skill_name.tmpl: {{printf "%q" .CatalogTitle}}
}

// ActivateSkillNameDescription renders the activate_skill name parameter description.
//
// Used by intent.activateSkillDefinition → ToolDefinition.Parameters.properties.name.description.
func ActivateSkillNameDescription(data ActivateSkillNameDescriptionData) string {
	return strings.TrimSpace(render("activate_skill_name.tmpl", data))
}

// KnowledgeEntryData is input for templates/knowledge_entry.tmpl.
type KnowledgeEntryData struct {
	Name    string // templates/knowledge_entry.tmpl: {{if .Name}}[{{.Name}}]{{end}}
	Content string // templates/knowledge_entry.tmpl: {{.Content}}
	Source  string // templates/knowledge_entry.tmpl: {{if .Source}} (source: {{.Source}}){{end}}
}

// KnowledgeEntry renders a static or retrieved knowledge entry.
//
// Used by runtime.formatKnowledge → buildContext static and RAG messages.
func KnowledgeEntry(data KnowledgeEntryData) string {
	return strings.TrimSpace(render("knowledge_entry.tmpl", data))
}

// ToolLockedData is input for templates/tool_locked.tmpl.
type ToolLockedData struct {
	ToolName          string // templates/tool_locked.tmpl: {{printf "%q" .ToolName}}
	SkillName         string // templates/tool_locked.tmpl: {{printf "%q" .SkillName}}
	SkillDescription  string // templates/tool_locked.tmpl: {{.SkillDescription}}
	ActivateSkillTool string // templates/tool_locked.tmpl: {{.ActivateSkillTool}}
}

// ToolLocked renders the message when a gated tool is called before activation.
//
// Used by runtime.handleToolCall when a skill-scoped tool is locked.
func ToolLocked(data ToolLockedData) string {
	return strings.TrimSpace(render("tool_locked.tmpl", data))
}
