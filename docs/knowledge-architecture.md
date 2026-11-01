# Knowledge base architecture for the ronyup MCP server

## Status

Proposed - March 2026

## Problem

The MCP server stores all knowledge in `ronyup/cmd/mcp/strings.go`. This knowledge includes instructions, tool descriptions, architecture hints, package metadata, characteristic guidance, plan steps, and file purposes. Go constants and function returns contain this content.

A documentation change requires a Go source edit and a new build. Non-Go contributors cannot improve the AI assistant guidance without changing Go code.

## Goals

1. Move knowledge to structured Markdown files under `ronyup/knowledge/`.
2. Embed the files with `go:embed` at compile time. Keep startup fast and distribute one binary.
3. Parse frontmatter and body content at server initialization. Produce typed knowledge structs.
4. Expose knowledge as MCP resources and prompts together with tools.
5. Move codegen templates in `implement_service.go` to `.gotmpl` files.

## Constraints

- Parse knowledge one time during initialization and store it in memory.
- Do not read from disk at runtime.
- Do not reload knowledge at runtime.

## 1. Directory layout

Store all knowledge inside `ronyup/`. This location uses the same Go module and simplifies `go:embed`.

```
ronyup/
├── knowledge/                          # Markdown knowledge files
│   ├── server/
│   │   └── instructions.md             # Server-level instructions sent in MCP init
│   │
│   ├── tools/
│   │   ├── list_templates.md           # Tool description + extended usage guidance
│   │   ├── read_template.md
│   │   ├── create_workspace.md
│   │   ├── create_feature.md
│   │   ├── plan_service.md
│   │   └── implement_service.md
│   │
│   ├── packages/                       # One file per x/ toolkit package
│   │   ├── di.md
│   │   ├── settings.md
│   │   ├── logkit.md
│   │   ├── tracekit.md
│   │   ├── meterkit.md
│   │   ├── testkit.md
│   │   ├── i18n.md
│   │   ├── apidoc.md
│   │   ├── cache.md
│   │   └── rkit.md
│   │
│   ├── architecture/                   # One file per architecture hint
│   │   ├── thin-handlers.md
│   │   ├── repo-ports.md
│   │   ├── postgres-sqlc.md
│   │   ├── gen-stub.md
│   │   ├── shared-pkg.md
│   │   ├── di-wiring.md
│   │   ├── settings-config.md
│   │   ├── logging.md
│   │   ├── tracing-metrics.md
│   │   ├── integration-tests.md
│   │   ├── apidoc-generation.md
│   │   ├── caching.md
│   │   ├── i18n-bundles.md
│   │   └── rkit-helpers.md
│   │
│   ├── characteristics/                # One file per service characteristic
│   │   ├── database.md
│   │   ├── cache.md
│   │   ├── api.md
│   │   ├── idempotent.md
│   │   ├── telemetry.md
│   │   ├── i18n.md
│   │   ├── testing.md
│   │   └── di.md
│   │
│   ├── planning/
│   │   ├── next-steps.md               # Ordered post-plan action items
│   │   └── file-purposes.md            # Purpose description per generated file type
│   │
│   └── prompts/                        # MCP prompt templates
│       ├── plan-service.md
│       └── review-architecture.md
│
├── internal/
│   ├── embed.go                        # existing: //go:embed skeleton
│   └── knowledge/                      # NEW: loader package
│       ├── embed.go                    # //go:embed ../../knowledge
│       ├── loader.go
│       └── types.go
│
└── cmd/mcp/
    └── ...                             # tools read from *knowledge.Base
```

## 2. Markdown file conventions

Each knowledge file uses YAML frontmatter between `---` delimiters. The frontmatter contains structured metadata. The body contains the knowledge.

Some body sections have defined meanings. Examples are `## Usage Hint` and `## Extended Guidance`.

### 2.1 Server instructions: `server/instructions.md`

This file has no frontmatter. The full body becomes the MCP server `Instructions` string.

```markdown
RonyKIT scaffolding assistant. Follow layered service conventions: ...

IMPORTANT - Always use the RonyKIT x/ toolkit packages instead of
third-party or hand-rolled equivalents:
• x/di - Dependency injection ...
...
```

### 2.2 Tool descriptions: `tools/{name}.md`

```markdown
---
name: plan_service
---

Create a dry-run implementation plan for a new service feature
based on requested characteristics.

## Extended Guidance

When the user asks to add a new service, invoke this tool first
to produce a plan. Review the plan output before calling
implement_service.
```

- **`name`** is required. It must match the tool constant in Go.
- **Body before the first `##`** becomes the short `Tool.Description` value.
- **`## Extended Guidance`** is optional. The server exposes this section as an MCP resource. It is not part of the tool description.

### 2.3 Package documentation: `packages/{short_name}.md`

```markdown
---
import_path: github.com/clubpay/ronykit/x/di
short_name: di
---

Dependency injection glue for fx-based service registration,
stub providers, and infra param wiring.

## Usage Hint

Use di.RegisterService in module.go;
di.ProvideDBParams/ProvideRedisParams for infra;
di.StubProvider for inter-service stubs.
```

- **`import_path`** is required. It contains the full Go import path.
- **`short_name`** is required. Prose and matching logic use this name.
- **Body before `## Usage Hint`** contains the package description.
- **`## Usage Hint`** contains short usage guidance for plan output.

### 2.4 Architecture hints: `architecture/{slug}.md`

These files have no frontmatter. The full body of each file is one architecture hint string.

```markdown
Keep API handlers thin: validate input and delegate business
behavior to internal/app.
```

The filename slug identifies the hint. For example, `thin-handlers` identifies the hint and its MCP resource URI.

### 2.5 Characteristics: `characteristics/{name}.md`

```markdown
---
keywords:
  - postgres
  - mysql
  - sql
  - database
applies_to_files:
  - repo
  - migration
  - settings
---

Default to Postgres with sqlc-backed repo ports + v0 adapters;
update migrations/sqlc and settings.
Wire DB params with x/di.ProvideDBParams and manage connection
settings through x/settings.

## File-Level Hint

Add persistence contracts in repo ports and implement mappings
in v0 adapters. Wire DB params with x/di.ProvideDBParams.
```

- **`keywords`** is required. The matcher compares these strings with user-supplied characteristic names. Matching is fuzzy and case-insensitive.
- **`applies_to_files`** is required. These path fragments select planned files for the file-level hint.
- **Body before `## File-Level Hint`** contains the service-level characteristic hint.
- **`## File-Level Hint`** contains the hint for planned files with matching paths.

### 2.6 Planning: `planning/next-steps.md`

Use an ordered Markdown list. Each list item becomes one next-step string.

```markdown
1. Run create_feature (or execute setup_command) to scaffold files ...
2. Implement contracts in api/service.go; keep handlers thin ...
3. Implement business orchestration in internal/app ...
   ...
```

### 2.7 Planning: `planning/file-purposes.md`

Use a YAML frontmatter map. Each key is a purpose identifier in Go code.

```markdown
---
service_lifecycle: >-
  Service module lifecycle and registration with x/di.RegisterService.
module_wiring: >-
  Dependency graph wiring with x/di and uber/fx for app, api,
  x/settings, and infra.
api_contracts: >-
  Contract routes, input/output DTOs, and thin transport handlers.
  Use x/apidoc for generated OpenAPI docs.
app_usecases: >-
  Application use-cases and business orchestration logic.
  Log with x/telemetry/logkit; use x/rkit helpers.
repo_ports: >-
  Repository interfaces consumed by app layer.
repo_adapters: >-
  Concrete datasource adapters implementing repository ports.
  Wire DB/Redis params with x/di.
migration: >-
  Database migration bundle bootstrap.
settings: >-
  Runtime settings model using x/settings (Viper-backed) with
  struct tags for DB/Redis/cache and module config.
---
```

This file needs no body. The frontmatter contains all data.

### 2.8 Prompt templates: `prompts/{name}.md`

```markdown
---
name: plan-service
description: Guide an AI agent through planning a new RonyKIT service feature.
arguments:
  - name: feature_name
    description: The name of the service feature to plan.
    required: true
  - name: characteristics
    description: Comma-separated list of characteristics (e.g. postgres, redis, rest-api).
    required: false
---

You are planning a new RonyKIT service feature called "{{feature_name}}".

Follow these steps:

1. Call the `plan_service` tool with the feature name and characteristics.
2. Review the architecture hints and recommended packages in the output.
3. Call `implement_service` to generate starter code.
4. Follow the next_steps from the plan output.

{{#if characteristics}}
Requested characteristics: {{characteristics}}
{{/if}}
```

- **`name`** is required. It contains the MCP prompt name.
- **`description`** is required. `prompts/list` displays this value.
- **`arguments`** is optional. It contains MCP `PromptArgument` definitions.
- **Body** contains the prompt template and `{{arg_name}}` placeholders.

## 3. Embedding and loader: `ronyup/internal/knowledge/`

The knowledge files and loader use the same Go module. One `go:embed` directive is sufficient. No module or workspace changes are necessary.

**`ronyup/internal/knowledge/embed.go`**:

```go
package knowledge

import "embed"

//go:embed all:../../knowledge
var content embed.FS
```

The `content` variable is unexported. External code calls `Load()`. This function reads `content` and returns the parsed `*Base`.

### 4.1 Types

```go
package knowledge

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

type ToolDoc struct {
	Name             string
	Description      string // body text before ## Extended Guidance
	ExtendedGuidance string // text under ## Extended Guidance (optional)
}

type PackageDoc struct {
	ImportPath  string `yaml:"import_path"`
	ShortName   string `yaml:"short_name"`
	Description string // body before ## Usage Hint
	UsageHint   string // text under ## Usage Hint
}

type ArchitectureHint struct {
	Slug string // filename stem, e.g. "thin-handlers"
	Text string // entire file body
}

type CharacteristicDoc struct {
	Keywords       []string `yaml:"keywords"`
	AppliesToFiles []string `yaml:"applies_to_files"`
	ServiceHint    string   // body before ## File-Level Hint
	FileHint       string   // text under ## File-Level Hint
}

type PromptDoc struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Arguments   []PromptArgument `yaml:"arguments"`
	Template    string           // body
}

type PromptArgument struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
}
```

### 4.2 Loader

```go
func Load() (*Base, error)
```

The loader follows these rules:

- Read from the package-level `content` embed.FS.
- Provide `LoadFS(fsys fs.FS) (*Base, error)` for unit tests with mock file systems.
- Call `Load()` one time at server startup.
- Walk each subdirectory and read each file.
- Separate frontmatter from the body.
- Parse YAML frontmatter into the correct struct.
- Extract named `##` sections from the body.
- Return `*Base`. Return an error if a required file is malformed.

### 4.3 Section parsing

A small helper splits a Markdown body at `## ` headings:

```go
func splitSections(body string) map[string]string
```

Example result:

```
{
  "":                 "text before first heading",
  "Usage Hint":       "text under ## Usage Hint",
  "File-Level Hint":  "text under ## File-Level Hint",
  "Extended Guidance": "text under ## Extended Guidance",
}
```

The empty-string key contains the preamble before the first heading.

## 5. Integration with the MCP server

### 5.1 Server configuration

Add the knowledge base to `serverConfig`:

```go
type serverConfig struct {
	name         string
	version      string
	instructions string // overridable with --instructions flag
	executable   string
	skeletonFS   fs.FS
	cmdRunner    runner
	kb           *knowledge.Base // NEW
}
```

### 5.2 Server initialization in `mcp.go`

```go
import "github.com/clubpay/ronykit/ronyup/internal/knowledge"

// In Cmd.RunE:
base, err := knowledge.Load()
if err != nil {
return fmt.Errorf("load knowledge base: %w", err)
}

instructions := opt.Instructions
if instructions == "" {
instructions = base.ServerInstructions
}

server := newServer(serverConfig{
...
instructions: instructions,
kb:           base,
})
```

### 5.3 Tool registration changes

Each tool registrar reads `cfg.kb` instead of constants:

```go
// Before:
mcpsdk.AddTool(srv, &mcpsdk.Tool{
	Name:        toolPlanService,
	Description: descPlanService,
}, handler)

// After:
toolDoc := cfg.kb.Tools["plan_service"]
mcpsdk.AddTool(srv, &mcpsdk.Tool{
	Name:        toolDoc.Name,
	Description: toolDoc.Description,
}, handler)
```

`characteristicHints()` and `fileHints()` read `cfg.kb.Characteristics`. They keep the current keyword matching logic. Frontmatter `keywords` and `applies_to_files` replace hardcoded switch cases.

### 5.4 MCP resources

Register each knowledge category as browsable resources. AI agents can then read knowledge on demand without tool calls.

**Static resources**, one for each known file:

```
knowledge://ronyup/packages/di
knowledge://ronyup/packages/settings
knowledge://ronyup/architecture/thin-handlers
knowledge://ronyup/characteristics/database
...
```

**Resource template** for dynamic browsing:

```
knowledge://ronyup/{category}/{name}
```

Register resources in `newServer()`:

```go
func registerResources(srv *mcpsdk.Server, kb *knowledge.Base) {
	// Register a resource template for browsing
	srv.AddResourceTemplate(
		&mcpsdk.ResourceTemplate{
			URITemplate: "knowledge://ronyup/{category}/{name}",
			Name:        "RonyKIT Knowledge Base",
			Description: "Architecture hints, package docs, and characteristic " +
				"guidance for RonyKIT service development.",
			MIMEType: "text/markdown",
		},
		resourceHandler(kb),
	)

	// Register individual static resources for discoverability
	for _, pkg := range kb.Packages {
		srv.AddResource(&mcpsdk.Resource{
			URI:         "knowledge://ronyup/packages/" + pkg.ShortName,
			Name:        "Package: " + pkg.ShortName,
			Description: pkg.Description,
			MIMEType:    "text/markdown",
		}, resourceHandler(kb))
	}

	for _, hint := range kb.ArchitectureHints {
		srv.AddResource(&mcpsdk.Resource{
			URI:         "knowledge://ronyup/architecture/" + hint.Slug,
			Name:        "Architecture: " + hint.Slug,
			Description: truncate(hint.Text, 100),
			MIMEType:    "text/markdown",
		}, resourceHandler(kb))
	}
	// ... characteristics, planning, etc.
}
```

The resource handler parses the URI. It finds the matching entry in `*Base` and returns the full Markdown content.

### 5.5 MCP prompts

Register prompt templates from `kb.Prompts`:

```go
func registerPrompts(srv *mcpsdk.Server, kb *knowledge.Base) {
	for _, p := range kb.Prompts {
		prompt := p // capture
		srv.AddPrompt(
			&mcpsdk.Prompt{
				Name:        prompt.Name,
				Description: prompt.Description,
				Arguments:   toMCPArguments(prompt.Arguments),
			},
			func(ctx context.Context, req *mcpsdk.GetPromptRequest) (
				*mcpsdk.GetPromptResult, error,
			) {
				rendered := renderTemplate(prompt.Template, req.Params.Arguments)
				return &mcpsdk.GetPromptResult{
					Description: prompt.Description,
					Messages: []mcpsdk.PromptMessage{
						{
							Role: mcpsdk.RoleUser,
							Content: &mcpsdk.TextContent{
								Text: rendered,
							},
						},
					},
				}, nil
			},
		)
	}
}
```

## 6. Codegen templates

The `implement_service.go` functions currently use `fmt.Sprintf` with raw Go source strings. These functions are `buildAPIServiceContent`, `buildAppServiceContent`, and `buildRepoPortContent`.

Move these source strings to `.gotmpl` files. Use `text/template`, as the current skeleton templates do.

These files are not knowledge files. They generate Go source code instead of supplying AI guidance. Store them beside the skeleton:

```
ronyup/internal/skeleton/codegen/
├── api_service.gotmpl
├── app_service.gotmpl
└── repo_port.gotmpl
```

The current `//go:embed skeleton` directive in `ronyup/internal/embed.go` embeds these files. The implement tool loads them with `text/template`.

This codegen change is separate from the knowledge base change. Both changes move content out of Go source.

## 7. Data flow diagram

```
┌──────────────────────────────────────────────────────────────────┐
│                  ronyup/knowledge/ (MD files)                    │
│                                                                  │
│  server/    tools/    packages/    architecture/                  │
│  characteristics/    planning/    prompts/                        │
└──────────────────────────┬───────────────────────────────────────┘
                           │
              go:embed all:../../knowledge
                           │
                           ▼
              ┌─────────────────────────────┐
              │  ronyup/internal/knowledge/ │
              │                             │
              │  embed.go  (var content FS) │
              │  loader.go (Load → *Base)   │
              │  types.go  (Base, etc.)     │
              └──────────────┬──────────────┘
                             │  called once at startup
                             ▼
              ┌─────────────────────────────┐
              │     serverConfig.kb         │
              │     (*knowledge.Base)       │
              └──────┬──────┬──────┬────────┘
                     │      │      │
         ┌───────────┘      │      └───────────┐
         ▼                  ▼                   ▼
   ┌───────────┐   ┌──────────────┐   ┌──────────────┐
   │   Tools   │   │  Resources   │   │   Prompts    │
   │           │   │              │   │              │
   │ plan_svc  │   │ knowledge:// │   │ plan-service │
   │ impl_svc  │   │ ronyup/...  │   │ review-arch  │
   │ etc.      │   │              │   │              │
   └───────────┘   └──────────────┘   └──────────────┘

   Tool handlers      AI agents        AI agents
   read kb fields     browse/read      invoke prompts
   for hints,         MD content       with arguments
   descriptions,      on demand        for guided
   packages, etc.                      workflows
```

## 8. Content that stays in Go code

| Category | Stays in Go? | Reason |
|----------|--------------|--------|
| Tool name constants such as `toolPlanService` | Yes | Code identifiers, not prose |
| Error format strings such as `errPathRequired` | Yes | Developer-facing text tied to code logic |
| Result format strings such as `msgPlannedFiles` | Yes | `fmt.Sprintf` format strings |
| Domain constants such as `ronyKitModulePath` | Yes | Structural constants |
| Keyword matching logic | Yes | Algorithm. Markdown frontmatter supplies the keywords. |
| File-path matching logic | Yes | Algorithm. Markdown supplies `applies_to_files`. |
| `toolkitPackage` struct type | Yes | Data type populated from Markdown |

## 9. Migration path

### Phase 1: Knowledge base foundation

1. Create `ronyup/knowledge/` with all Markdown files from `strings.go`.
2. Create `ronyup/internal/knowledge/` with `embed.go`, `types.go`, and `loader.go`.
3. Call `Load()` from `newServer()`.
4. Replace `strings.go` constants with `kb.*` reads.
5. Delete `strings.go` constants that moved to Markdown files.

### Phase 2: MCP resources and prompts

7. Register MCP resources from the knowledge base.
8. Write prompt template Markdown files.
9. Register MCP prompts from the knowledge base.

### Phase 3: Codegen templates

10. Convert `buildAPIServiceContent` to `api_service.gotmpl`.
11. Convert `buildAppServiceContent` to `app_service.gotmpl`.
12. Convert `buildRepoPortContent` to `repo_port.gotmpl`.
13. Update `implement_service.go` to load and execute templates.

## 10. Add new knowledge

### Add a toolkit package

This example adds `x/newpkg`:

1. Create `ronyup/knowledge/packages/newpkg.md` with frontmatter.
2. Rebuild `ronyup`.
3. The MCP server then does these operations:
   - Includes the package in `plan_service` recommendations.
   - Exposes `knowledge://ronyup/packages/newpkg` as a resource.
   - References the package in server instructions if `instructions.md` includes it.

### Add a characteristic

This example adds `graphql`:

1. Create `ronyup/knowledge/characteristics/graphql.md`.
2. Set `keywords: [graphql, gql]` and `applies_to_files: [api]`.
3. Rebuild `ronyup`.

The keyword matcher then finds the characteristic. No Go code change is necessary.

### Add an architecture hint

1. Create `ronyup/knowledge/architecture/new-hint.md`.
2. Rebuild `ronyup`.

The server includes the hint in `plan_service` output and exposes it as a resource.

### Add an MCP prompt

1. Create `ronyup/knowledge/prompts/new-prompt.md` with frontmatter.
2. Rebuild `ronyup`.

The prompt becomes available through `prompts/list` and `prompts/get`.
