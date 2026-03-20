# Knowledge Base Architecture for ronyup MCP Server

## Status

Proposed — March 2026

## Problem

All MCP server knowledge — instructions, tool descriptions, architecture hints,
package metadata, characteristic guidance, plan steps, file purposes — is
hardcoded in `ronyup/cmd/mcp/strings.go` as Go constants and function returns.
Improving any hint or adding a new toolkit package requires editing Go source and
recompiling. There is no way for non-Go contributors to improve the AI
assistant's quality.

## Goals

1. Move knowledge into structured **Markdown files** under `ronyup/knowledge/`.
2. Embed them via `go:embed` at compile time (fast startup, single binary).
3. Parse frontmatter + body at server init to produce typed knowledge structs.
4. Expose knowledge as MCP **resources** and **prompts** alongside tools.
5. Make the codegen templates in `implement_service.go` use `.gotmpl` files.

## Constraints

- Startup must remain fast: parse once at init, store in memory.
- No runtime disk I/O or hot-reload.

---

## 1. Directory Layout

All knowledge lives inside `ronyup/` — same module, simple `go:embed`.

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

---

## 2. Markdown File Conventions

Every knowledge file uses **YAML frontmatter** (delimited by `---`) for
structured metadata. The **body** is the knowledge content. Specific sections
within the body carry semantic meaning (e.g., `## Usage Hint`).

### 2.1 Server Instructions — `server/instructions.md`

No frontmatter. The entire file body becomes the MCP server `Instructions`
string.

```markdown
RonyKIT scaffolding assistant. Follow layered service conventions: ...

IMPORTANT — Always use the RonyKIT x/ toolkit packages instead of
third-party or hand-rolled equivalents:
  • x/di — Dependency injection ...
  ...
```

### 2.2 Tool Descriptions — `tools/{name}.md`

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

- **`name`** (required): Must match the tool constant in Go.
- **Body before first `##`**: Short description used as `Tool.Description`.
- **`## Extended Guidance`** (optional): Surfaced as an MCP resource, not
  part of the tool description.

### 2.3 Package Docs — `packages/{short_name}.md`

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

- **`import_path`** (required): Full Go import path.
- **`short_name`** (required): The name used in prose and matching.
- **Body before `## Usage Hint`**: Package description.
- **`## Usage Hint`**: Concise usage guidance returned in plan output.

### 2.4 Architecture Hints — `architecture/{slug}.md`

No frontmatter. Each file's entire body is one architecture hint string.

```markdown
Keep API handlers thin: validate input and delegate business
behavior to internal/app.
```

The filename slug (e.g., `thin-handlers`) is used as the hint identifier
and for MCP resource URIs.

### 2.5 Characteristics — `characteristics/{name}.md`

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
Wire DB params via x/di.ProvideDBParams and manage connection
settings through x/settings.

## File-Level Hint

Add persistence contracts in repo ports and implement mappings
in v0 adapters. Wire DB params via x/di.ProvideDBParams.
```

- **`keywords`** (required): Strings matched against user-provided
  characteristic names (fuzzy, case-insensitive).
- **`applies_to_files`** (required): Path fragments that determine which
  planned files receive the file-level hint.
- **Body before `## File-Level Hint`**: Service-level characteristic hint.
- **`## File-Level Hint`**: Hint applied to individual planned files whose
  path matches `applies_to_files`.

### 2.6 Planning — `planning/next-steps.md`

Ordered Markdown list. Each list item becomes one next-step string.

```markdown
1. Run create_feature (or execute setup_command) to scaffold files ...
2. Implement contracts in api/service.go; keep handlers thin ...
3. Implement business orchestration in internal/app ...
...
```

### 2.7 Planning — `planning/file-purposes.md`

YAML frontmatter map. Keys are purpose identifiers used in Go code.

```markdown
---
service_lifecycle: >-
  Service module lifecycle and registration via x/di.RegisterService.
module_wiring: >-
  Dependency graph wiring via x/di and uber/fx for app, api,
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
  Wire DB/Redis params via x/di.
migration: >-
  Database migration bundle bootstrap.
settings: >-
  Runtime settings model using x/settings (Viper-backed) with
  struct tags for DB/Redis/cache and module config.
---
```

No body needed — all data is in frontmatter.

### 2.8 Prompt Templates — `prompts/{name}.md`

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

- **`name`** (required): MCP prompt name.
- **`description`** (required): Shown in `prompts/list`.
- **`arguments`** (optional): List of MCP `PromptArgument` definitions.
- **Body**: The prompt template with `{{arg_name}}` placeholders.

---

## 3. Embedding & Loader — `ronyup/internal/knowledge/`

Since knowledge files live inside `ronyup/` (same Go module), embedding
is a single `go:embed` directive — no extra modules, no workspace changes.

**`ronyup/internal/knowledge/embed.go`**:

```go
package knowledge

import "embed"

//go:embed all:../../knowledge
var content embed.FS
```

The `content` var is unexported. External code calls `Load()` which
reads from it and returns the parsed `*Base`.

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
    Description      string   // body text before ## Extended Guidance
    ExtendedGuidance string   // text under ## Extended Guidance (optional)
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
    Name        string            `yaml:"name"`
    Description string            `yaml:"description"`
    Arguments   []PromptArgument  `yaml:"arguments"`
    Template    string            // body
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

- Reads from the package-level `content` embed.FS.
- A `LoadFS(fsys fs.FS) (*Base, error)` variant is also provided for
  unit testing with mock file systems.
- Called **once** at server startup.
- Walks each subdirectory, reads files, splits frontmatter from body,
  parses YAML frontmatter into the corresponding struct, extracts
  named `##` sections from the body.
- Returns `*Base` or an error if any required file is malformed.

### 4.3 Section Parsing

A small helper splits a Markdown body by `## ` headings:

```go
func splitSections(body string) map[string]string
```

Returns a map like:
```
{
  "":                 "text before first heading",
  "Usage Hint":       "text under ## Usage Hint",
  "File-Level Hint":  "text under ## File-Level Hint",
  "Extended Guidance": "text under ## Extended Guidance",
}
```

The empty-string key holds the "preamble" text before any heading.

---

## 5. Integration with MCP Server

### 5.1 Server Config

Add the knowledge base to `serverConfig`:

```go
type serverConfig struct {
    name         string
    version      string
    instructions string  // overridable via --instructions flag
    executable   string
    skeletonFS   fs.FS
    cmdRunner    runner
    kb           *knowledge.Base  // NEW
}
```

### 5.2 Server Init (`mcp.go`)

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

### 5.3 Tool Registration Changes

Each tool registrar reads from `cfg.kb` instead of constants:

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

`characteristicHints()` and `fileHints()` use `cfg.kb.Characteristics`
with the same keyword matching logic, but now driven by the `keywords`
and `applies_to_files` from frontmatter instead of hardcoded switch cases.

### 5.4 MCP Resources

Register each knowledge category as browsable resources so AI agents can
read them on demand without calling tools.

**Static resources** (one per known file):

```
knowledge://ronyup/packages/di
knowledge://ronyup/packages/settings
knowledge://ronyup/architecture/thin-handlers
knowledge://ronyup/characteristics/database
...
```

**Resource template** (for dynamic browsing):

```
knowledge://ronyup/{category}/{name}
```

Registration in `newServer()`:

```go
func registerResources(srv *mcpsdk.Server, kb *knowledge.Base) {
    // Register a resource template for browsing
    srv.AddResourceTemplate(
        &mcpsdk.ResourceTemplate{
            URITemplate: "knowledge://ronyup/{category}/{name}",
            Name:        "RonyKIT Knowledge Base",
            Description: "Architecture hints, package docs, and characteristic " +
                "guidance for RonyKIT service development.",
            MIMEType:    "text/markdown",
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

The resource handler parses the URI, looks up the matching entry in `*Base`,
and returns the full Markdown content.

### 5.5 MCP Prompts

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

---

## 6. Codegen Templates

The `implement_service.go` functions (`buildAPIServiceContent`,
`buildAppServiceContent`, `buildRepoPortContent`) currently use
`fmt.Sprintf` with raw Go source strings. These should be converted to
`.gotmpl` files using `text/template`, following the same pattern as the
existing skeleton templates.

These are **not** knowledge files (they don't teach the AI; they produce
Go source code). They belong alongside the skeleton:

```
ronyup/internal/skeleton/codegen/
├── api_service.gotmpl
├── app_service.gotmpl
└── repo_port.gotmpl
```

They are embedded by the existing `//go:embed skeleton` directive in
`ronyup/internal/embed.go` and loaded via `text/template` in the
implement tool. This is a separate refactoring from the knowledge base
but follows the same "externalize from Go code" principle.

---

## 7. Data Flow Diagram

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

---

## 8. What Stays in Go Code

| Category | Stays in Go? | Reason |
|---|---|---|
| Tool name constants (`toolPlanService`) | Yes | Code identifiers, not prose |
| Error format strings (`errPathRequired`) | Yes | Developer-facing, tied to code logic |
| Result format strings (`msgPlannedFiles`) | Yes | `fmt.Sprintf` format strings |
| Domain constants (`ronyKitModulePath`) | Yes | Structural constants |
| Keyword matching logic | Yes | Algorithm, but keywords come from MD frontmatter |
| File-path matching logic | Yes | Algorithm, but `applies_to_files` comes from MD |
| `toolkitPackage` struct type | Yes | Data type (populated from MD now) |

---

## 9. Migration Path

### Phase 1 — Knowledge Base Foundation
1. Create `ronyup/knowledge/` directory with all MD files (migrate from `strings.go`).
2. Create `ronyup/internal/knowledge/` with `embed.go`, `types.go`, `loader.go`.
3. Wire `Load()` into `newServer()`.
4. Replace `strings.go` constants with `kb.*` reads.
5. Delete `strings.go` constants that are now in MD files.

### Phase 2 — MCP Resources & Prompts
7. Register MCP resources from knowledge base.
8. Write prompt template MD files.
9. Register MCP prompts from knowledge base.

### Phase 3 — Codegen Templates
10. Convert `buildAPIServiceContent` → `api_service.gotmpl`.
11. Convert `buildAppServiceContent` → `app_service.gotmpl`.
12. Convert `buildRepoPortContent` → `repo_port.gotmpl`.
13. Update `implement_service.go` to load and execute templates.

---

## 10. Adding New Knowledge

To add a new toolkit package (e.g., `x/newpkg`):

1. Create `ronyup/knowledge/packages/newpkg.md` with frontmatter.
2. Rebuild `ronyup`.
3. The MCP server automatically:
   - Includes it in `plan_service` recommended packages.
   - Exposes it as `knowledge://ronyup/packages/newpkg` resource.
   - References it in server instructions (if `instructions.md` is updated).

To add a new characteristic (e.g., `graphql`):

1. Create `ronyup/knowledge/characteristics/graphql.md` with
   `keywords: [graphql, gql]` and `applies_to_files: [api]`.
2. Rebuild.
3. The keyword matcher automatically picks it up — no Go code changes.

To add a new architecture hint:

1. Create `ronyup/knowledge/architecture/new-hint.md`.
2. Rebuild.
3. Automatically included in `plan_service` output and available as a resource.

To add a new MCP prompt:

1. Create `ronyup/knowledge/prompts/new-prompt.md` with frontmatter.
2. Rebuild.
3. Available via `prompts/list` and `prompts/get`.
