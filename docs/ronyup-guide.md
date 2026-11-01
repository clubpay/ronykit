# ronyup: Scaffolding CLI and MCP server

`ronyup` is a CLI that scaffolds RonyKIT projects and features. It also runs as an MCP server. AI assistants can use this server for scaffolding and code generation.

## Table of contents

- [Installation](#installation)
- [Create a workspace](#create-a-workspace)
- [Add a feature](#add-a-feature)
- [Executable bundles](#executable-bundles)
- [Upgrade an older workspace](#upgrade-an-older-workspace)
- [Feature templates](#feature-templates)
- [Translation catalogs](#translation-catalogs)
- [Lint: x/rkit duplicates](#lint-xrkit-duplicates)
- [MCP server](#mcp-server)
  - [Setup for Cursor IDE](#setup-for-cursor-ide)
  - [Setup for JetBrains GoLand](#setup-for-jetbrains-goland)
  - [Available MCP tools](#available-mcp-tools)
  - [Workflow example](#workflow-example)
  - [MCP conventions](#mcp-conventions)
- [Interactive mode](#interactive-mode)
- [Command reference](#command-reference)

## Installation

Install with [Homebrew](https://brew.sh/) on macOS or Linux:

```bash
brew install clubpay/tap/ronyup
```

This command taps `clubpay/homebrew-tap`. It installs a prebuilt `ronyup` binary for macOS or Linux on Intel or ARM.

[Homebrew 6 and later](https://docs.brew.sh/Tap-Trust) requires explicit trust for third-party taps. The fully qualified install command handles this requirement. Upgrade with `brew upgrade clubpay/tap/ronyup`.

If you tapped `clubpay/tap` separately, run this command before installation by short name:

```bash
brew trust --formula clubpay/tap/ronyup
```

[GitHub Releases](https://github.com/clubpay/ronykit/releases) also provides prebuilt binaries. Tags use the form `ronyup/vX.Y.Z`. Archives are available for macOS, Linux, and Windows.

You can also install with the Go toolchain:

```bash
go install github.com/clubpay/ronykit/ronyup@latest
```

Verify the installation:

```bash
ronyup --help
```

## Create a workspace

Create a RonyKIT project with a complete workspace:

```bash
ronyup setup workspace \
    --repoDir ./my-api \
    --repoModule github.com/you/my-api
```

The command creates this structure:

```
my-api/
├── go.work              # Go workspace file
├── bundles.yaml         # executable bundle manifest
├── Makefile
├── pkg/runner/     # shared bootstrap for all executables
├── cmd/all-in-one/         # default all-in-one dev entry point
├── pkg/i18n/            # Translation utilities
├── devops/              # devbox: Helm-managed K8s services
├── docs/                # design docs and guides
└── .git/                # Initialized git repo
```

Run the workspace with `make run`. You can also run `cd cmd/all-in-one && go run .`.

### Workspace kinds

Use `--kind` to select the layout:

- **`backend`** is the default. It creates a Go-only workspace at `repoDir`.
- **`fullstack`** creates separate `backend/` and `frontend/` directories:

```
my-api/
├── backend/          # the Go workspace (go.work, cmd/all-in-one, pkg/, feature/, Makefile, .golangci.yml)
├── frontend/         # web/mobile application (framework-agnostic placeholder)
├── devops/           # devbox: Helmfile-managed K8s services (shared)
├── docs/             # design docs and guides (shared)
├── AGENTS.md         # agent guidance (shared)
└── .ai/ .agents/ .cursor/   # AI assistant config (shared)
```

In `fullstack` mode, Go module paths start with `backend/`. For example: `github.com/you/my-api/backend/cmd/all-in-one`.

Run `ronyup setup feature`, `make`, and `go` commands from `backend/`.

```bash
ronyup setup workspace \
    --repoDir ./my-api \
    --repoModule github.com/you/my-api \
    --kind fullstack
```

### Pre-installed agent skills

Every workspace includes the `ronykit-framework` skill. `setup workspace` can also install general-purpose agent skills under `.agents/skills/`. These skills cover code style, testing, formatting, debugging, review, and commits.

Cursor and Codex read `.agents/skills/`. Claude Code does not discover this directory. The scaffold creates `.claude/skills/<id>` as a symlink to the same skill. Scaffolds do not copy skills to `.cursor/skills/`.

Use `--skills` or `-s` to select skills. Specify a comma-separated list of skill IDs. You can also use `default`, `all`, or `none`:

```bash
# install the defaults for the chosen kind (same as omitting the flag)
ronyup setup workspace -r ./my-api -m github.com/you/my-api

# install everything
ronyup setup workspace -r ./my-api -m github.com/you/my-api --skills all

# pick a specific set
ronyup setup workspace -r ./my-api -m github.com/you/my-api \
    --skills go-modern,go-testing,systematic-debugging

# install none
ronyup setup workspace -r ./my-api -m github.com/you/my-api --skills none
```

Run `ronyup setup` without flags for an interactive selector. The selector preselects the defaults for the chosen kind.

| Skill ID | Category | Default | Description |
|----------|----------|---------|-------------|
| `go-modern` | Go | backend, fullstack | Idiomatic Go 1.25+/1.26, including generics, iterators, errors, and context |
| `go-testing` | Go | backend, fullstack | Table-driven testify tests, required x/testkit repository integration tests, and app unit tests |
| `go-design` | Go | backend, fullstack | Dependency direction, consumer-defined interfaces, composition, safe refactoring, and untested code |
| `release-it` | Architecture | backend, fullstack | Timeouts, circuit breakers, bulkheads, health checks, and observability |
| `domain-driven-design` | Architecture | backend, fullstack | Bounded contexts, aggregates, and value objects mapped to feature modules |
| `software-design-philosophy` | Architecture | backend, fullstack | Deep modules, information hiding, and removal of pass-through layers |
| `ddia-systems` | Architecture | opt-in | Storage, indexes, partitioning, isolation, and idempotency |
| `writing-tests` | Quality | all kinds | Language-independent TDD discipline and the test pyramid |
| `code-formatting` | Quality | all kinds | Formatters, linters, and clean diffs |
| `systematic-debugging` | Quality | all kinds | Root-cause analysis, reproduction, verification, and prevention |
| `code-review` | Quality | all kinds | Review priorities for correctness, design, and risk |
| `verification-before-completion` | Quality | all kinds | Fresh evidence before completion claims |
| `writing-plans` | Workflow | all kinds | Small, testable tasks for multi-step work |
| `conventional-commits` | Workflow | all kinds | Atomic commits in the Conventional Commits format |
| `technical-documentation` | Workflow | all kinds | Google developer documentation style for READMEs, SRS/SDD prose, and doc comments |
| `nextjs-modern` | Frontend | fullstack, frontend | Next.js 16 App Router, Server Components, and Server Actions |
| `frontend-testing` | Frontend | fullstack, frontend | Vitest, Testing Library, and Playwright end-to-end practices |
| `storybook` | Frontend | fullstack, frontend | Storybook stories for UI components |
| `shadcn` | Frontend | fullstack, frontend | Find, install, compose, and theme shadcn/ui components |
| `dashboard-ui` | Frontend | fullstack, frontend | Dashboard and back-office surfaces with KPI strips, hierarchy, and tables |
| `frontend-design` | Frontend | fullstack, frontend | Required surface mode, design questions, and design review before UI bootstrap |
| `ux-quality` | Frontend | fullstack, frontend | WCAG AA keyboard and focus behavior, interaction states, forms, copy, and layout resilience |
| `ui-polish` | Frontend | fullstack, frontend | Surface, icon, microinteraction, animation, and motion performance guidance |
| `typography` | Frontend | fullstack, frontend | Font pairing, scales, line height, OpenType, and JSX entities |
| `design-tokens` | Frontend | fullstack, frontend | Three-tier OKLCH tokens, four-tier text, dark mode, Tailwind, and shadcn |
| `react-performance` | Frontend | fullstack, frontend | React and Next.js performance rules for waterfalls, bundles, and re-renders |
| `composition-patterns` | Frontend | fullstack, frontend | Compound components, lifted state, and explicit variants |
| `webmcp` | Frontend | fullstack, frontend | Frontend features exposed as agent-callable WebMCP tools |

The workspace `AGENTS.md` lists installed skills. Its skill routing table maps tasks to the `SKILL.md` files that agents must read.

### Agent enforcement gates

Scaffolded workspaces include automatic gates for design and testing:

| Gate | Enforcement | Checks |
|------|-------------|--------|
| Backend SRS/SDD | `scaffold_feature` MCP tool | Approved `docs/design/<feature>-{srs,sdd}.md` files exist before scaffolding |
| Frontend design document | `frontend/verify.sh` | Approved `docs/design/<app>-frontend-design.md` exists before UI verification passes |
| Backend repository and app tests | `verify.sh` and Cursor and Claude Code `backend-verify` Stop hooks | Each repository port method has an integration test. Each `App` method has a unit test. Tests pass. |
| Package selection for imports | `.golangci.yml` `depguard` through `verify.sh` | Forbidden imports are absent. Examples include `go.temporal.io/sdk`, logging packages, ID packages, `samber/lo`, strcase, and copier. |
| Package selection for code | `ronyup lint` through `verify.sh` and Claude Code `backend-lint` PostToolUse hook | Handwritten copies of `x/rkit` helpers are absent |
| Frontend quality | `frontend/verify.sh` and Cursor and Claude Code `frontend-verify` Stop hooks | Typecheck, lint, build, test, Storybook stories, and UI polish checks pass |

MCP prompt `design-frontend` coordinates frontend bootstrap. The sequence is questions, design document, approval, initialization, and implementation.

See `knowledge://ronyup/architecture/integration-tests` for backend integration test conventions.

> `verification-before-completion` and `writing-plans` are adapted from [obra/superpowers](https://github.com/obra/superpowers) under the MIT license.

### Workspace flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--repoDir` | `-r` | Destination directory | `./my-repo` |
| `--repoModule` | `-m` | Go module path | `github.com/your/repo` |
| `--appName` | `-a` | Application name | `myapp` |
| `--kind` | `-k` | Layout: `backend` or `fullstack` | `backend` |
| `--skills` | `-s` | Skills to install: IDs or `default`, `all`, `none` | Kind defaults |
| `--force` | `-f` | Clean the destination before setup | `false` |

## Add a feature

Add a service module to an existing workspace:

```bash
cd my-api
ronyup setup feature \
    --featureDir users \
    --featureName users \
    --template service \
    --repoModule github.com/you/my-api
```

The command creates a feature module under `feature/users/`. The module includes:

- API contracts and handler stubs
- App layer for business logic
- Repository ports and adapters with sqlc support
- Settings and configuration
- Dependency injection with uber/fx
- Migration setup
- Stub code generator

The command adds the feature to `go.work`. It also registers the feature in `cmd/all-in-one/features.go`.

The default feature path is `{featurePrefix}/{featureDir}/`, such as `feature/users/`. Use `--groupByTemplate` or `-g` for `{featurePrefix}/{template}/{featureDir}/`, such as `feature/service/users/`.

### Feature flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--featurePrefix` | | Parent directory for feature modules | `feature` |
| `--featureDir` | `-p` | Directory name inside the feature prefix | `my_feature` |
| `--featureName` | `-n` | Feature name | `myfeature` |
| `--template` | `-t` | Template: `service` only | `service` |
| `--groupByTemplate` | `-g` | Group under `{featurePrefix}/{template}/` | `false` |
| `--repoModule` | `-m` | Repository Go module path, automatically detected | `github.com/your/repo` |
| `--force` | `-f` | Replace the existing feature directory | `false` |

The command refreshes import lists for matching bundles in `bundles.yaml`.

## Executable bundles

Workspaces compile feature modules into one or more executables. The default `cmd/all-in-one` binary is the development entry point. Production bundles can contain different feature combinations at compile time.

```bash
ronyup setup bundle --name auth-api --services feature/auth,feature/session
ronyup setup bundle --gen
cd cmd/all-in-one && go run . --service feature/auth
```

See `bundles.yaml` at the Go workspace root. Available Makefile targets are `make run`, `make run-bundle BUNDLE=auth-api`, and `make build-bundle BUNDLE=auth-api`.

## Upgrade an older workspace

Upgrade `ronyup` first. Then refresh shared boilerplate and migrate the bundle layout:

```bash
ronyup setup sync
ronyup setup migrate bundles    # from repo root (fullstack) or backend/ - both work
ronyup setup sync --only backend --overwrite   # optional Makefile bundle targets
```

`setup sync` does not rewrite `cmd/all-in-one/main.go`. Use `setup migrate bundles`. Use `--dry-run` to preview the migration.

By default, `setup sync` adds only missing files. It does not change existing `AGENTS.md`, hooks, skill trees, or `Makefile`. Use `--overwrite` to replace these files:

```bash
ronyup setup sync --only agents,hooks --overwrite
ronyup setup sync --only skills --overwrite --skills installed
ronyup setup sync --only backend --overwrite
```

`--skills installed` refreshes skills under `.agents/skills/`. Legacy copies under `.cursor/skills/` or `.claude/skills/` also count as installed skills.

Sync replaces retired skills `clean-architecture`, `refactoring-patterns`, and `working-with-legacy-code` with `go-design`. With `--overwrite`, sync deletes the retired directories. Without `--overwrite`, sync only reports them.

This mode does not add new catalog skills. Use `--skills default` or specify a skill ID to add skills.

## Feature templates

| Template | Purpose |
|----------|---------|
| `service` | Standard API service with CRUD endpoints, repository layer, and dependency injection |

## Translation catalogs

Generate i18n translation catalogs with `golang.org/x/text/message/pipeline`:

```bash
ronyup text \
    --src-lang en-US \
    --dst-lang en-US,fa-IR,ar-AR \
    --out-dir . \
    --gen-package pkg/i18n \
    --gen-file catalog.go
```

If a `go.work` file exists, the command loads packages from each workspace module.

### Text flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--src-lang` | `-s` | Source language | `en-US` |
| `--dst-lang` | `-l` | Target languages | `en-US,fa-IR,ar-AR` |
| `--out-dir` | `-o` | Output directory | `.` |
| `--gen-package` | `-g` | Package path for the generated file | - |
| `--gen-file` | `-f` | Generated file name | `catalog.go` |
| `--modules-filter` | `-m` | Filter go.work modules | - |

## Lint: x/rkit duplicates

`ronyup lint` reports Go code that implements an existing `x/rkit` helper again. Use `--fix` to rewrite supported findings:

```bash
ronyup lint ./...          # report; exit 1 when findings remain, 2 when packages do not compile
ronyup lint --fix ./...    # rewrite fixable findings in place (adds the rkit import)
```

| Rule | Matches | Replacement |
|------|---------|-------------|
| `must` | `v, err := f(); if err != nil { panic(err) }` | `rkit.Must(f())` or `rkit.Assert(f())` |
| `coalesce` | `if x == "" { x = y }` statements and `if a != "" { return a }` chains | `rkit.Coalesce(...)` |
| `transform` | Loops that only map or filter a slice | `rkit.Map(s, f)` or `rkit.Filter(s, keep)` |
| `mapcollect` | A loop that only appends map keys or values | `rkit.MapKeysToArray(m)` or `rkit.MapToArray(m)` |
| `discard` | `v, _ := f()` or `v, _ := x.(T)` | `rkit.Ok` or `rkit.TryCast[T]` |
| `stdlib` | `slices.Contains`, `cmp.Or`, `strconv.Itoa`, `FormatInt(x, 10)`, `fmt.Sprintf("%d", x)` | `rkit.Contains`, `Coalesce`, `IntToStr`, and related helpers |
| `unsafeconv` | Manual zero-copy conversions between `[]byte` and `string` | `rkit.B2S` or `rkit.S2B` |
| `helper` | Unexported copies of `Must`, `PtrVal`, `Contains`, or `Coalesce` | Delete the helper and call rkit. No automatic fix is available. |

The linter skips modules whose `go.mod` does not require `x/rkit`. Use `--all-modules` to include these modules.

For a deliberate exception, add `//rkitcheck:ignore <reason>` on the reported line or the preceding line.

The MCP tool `check_rkit_usage` runs the same check. Scaffolded workspaces run it from `verify.sh`. Claude Code runs it after each Go edit with `.claude/hooks/backend-lint.sh`.

## MCP server

`ronyup` includes a [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server. It exposes scaffolding and code generation tools to AI assistants.

You can give requests such as these to an AI assistant:

- "Create a new user service with CRUD endpoints and Postgres storage."
- "Add a payment feature with idempotency support."
- "Plan the implementation of a notification service."

The assistant can use `ronyup` tools to scaffold and implement the code with RonyKIT conventions.

### Start the MCP server manually

```bash
ronyup mcp
```

The server is ready when it starts and waits for input.

### Setup for Cursor IDE

Create or update `.cursor/mcp.json` in the project root:

```json
{
	"mcpServers": {
		"ronyup": {
			"command": "ronyup",
			"args": ["mcp"]
		}
	}
}
```

Reopen Cursor or reload the window. The `ronyup` tools appear in the AI chat panel.

### Setup for JetBrains GoLand

1. Open **Settings**. On macOS, open **Preferences**.
2. Search for **MCP**.
3. Add a **stdio** MCP server with these values:
   - **Name:** `ronyup`
   - **Command:** `ronyup`
   - **Arguments:** `mcp`
4. Save the configuration. Restart GoLand if the tools do not appear.

### Setup for VS Code

Add this configuration to `.vscode/settings.json` or your user settings:

```json
{
	"mcp": {
		"servers": {
			"ronyup": {
				"command": "ronyup",
				"args": ["mcp"]
			}
		}
	}
}
```

### Available MCP tools

| Tool | Description |
|------|-------------|
| `scaffold_workspace` | Create a RonyKIT workspace at `path`. `kind` accepts `backend` or `fullstack`. Optional `skills` accepts IDs or `default`, `all`, `none`. |
| `scaffold_feature` | Add a `feature/<name>/` module to a workspace |
| `sync_workspace` | Refresh scaffold boilerplate in an existing workspace with `setup sync` |
| `sync_skills` | Sync skills from the embedded catalog and return validation results |
| `validate_skills` | Check skill content, frontmatter, links, and retirement without changes |
| `setup_bundle` | Create or remove executable bundles with `setup bundle` |
| `migrate_bundles` | Upgrade a legacy workspace to the bundle layout |
| `check_frontend_design` | Verify an approved and complete frontend design document before UI work |
| `check_rkit_usage` | Report and optionally fix code that implements an x/rkit helper again |

### MCP prompts

| Prompt | Description |
|--------|-------------|
| `design-new-service` | Full workflow: SRS, SDD, scaffold, and code |
| `write-srs` | Write an SRS to `docs/design/<feature>-srs.md` |
| `write-sdd` | Write an SDD from an approved SRS |
| `design-api` | Design contracts, routes, and handlers |
| `plan-service` | Plan a feature with the SRS/SDD-first workflow |
| `write-service-code` | Implement service code from the SDD |
| `write-workflow` | Write Temporal and `flow` workflows |
| `review-architecture` | Review feature layout for compliance |
| `generate-stubs` | Generate and use client stubs |
| `migrate-kit-to-rony` | Migrate incrementally from `kit` to `rony` |

### Knowledge resources

The server exposes Markdown resources for architecture, `x/` packages, characteristics, and tools. Resource URIs use `knowledge://ronyup/<category>/<name>`.

Use the MCP resource browser in your IDE. You can also see `.agents/skills/ronykit-framework/references/mcp-map.md` in scaffolded workspaces.

### Agent skill

Scaffolded workspaces include the **ronykit-framework** skill under `.agents/skills/`. The directory follows the [Agent Skills](https://agentskills.io/specification) layout.

Cursor and Codex discover this directory. `.claude/skills/ronykit-framework` is a symlink to the same files for Claude Code.

Invoke `/ronykit-framework` in Cursor for workflows that coordinate MCP tools, prompts, and resources. The skill refers to the knowledge base instead of duplicating it.

The skill includes the Elephant–Goldfish Model (EGM) protocol under `references/`. It also includes reviewer lenses and rubrics.

The Elephant drafts with project context. Fresh read-only Goldfish subagents review without the drafting conversation. The user decides findings and approval. No separate EGM plugin is necessary.

### Workflow example

Use this AI-assisted workflow for a new service feature:

1. **Write the requirements.** Use MCP prompt `write-srs` or phase 1 of `design-new-service`. Write `docs/design/<feature>-srs.md`. Run EGM zoom-out and comprehension reviews. Resolve findings with the user. Get approval before you continue.
2. **Write the design.** Use MCP prompt `write-sdd` or phase 2 of `design-new-service`. Write `docs/design/<feature>-sdd.md`. Run all five EGM stages. Resolve findings with the user. Get approval before scaffolding.
3. **Create the scaffold.** Call `scaffold_feature`. For a new repository, call `scaffold_workspace`.
4. **Load knowledge.** Read MCP architecture resources before implementation. These resources include `design-documents`, `egm-design-review`, `service-structure`, `api-handler-files`, and `repo-ports`.
5. **Plan and implement.** For multi-step work, run an EGM implementation-readiness review on the plan. Then use MCP prompt `write-service-code`. Follow the approved SDD.
6. **Iterate and verify.** Run `make gen-stub` after contract changes. Refine handlers and app logic. Run an EGM drift report for a separate design and code consistency audit. This report does not replace tests or code review.

### MCP conventions

The MCP server guides AI assistants to follow these conventions:

- Define persistence interfaces in `internal/repo/port.go`.
- Implement persistence adapters in `internal/repo/v0`.
- Use Postgres and sqlc as the default storage stack.
- Use a different storage stack only when the user explicitly requests it.
- Run `make gen-stub` after contract changes.
- Use generated stubs for inter-service calls. Do not write HTTP clients manually.
- Connect dependencies with `x/di` and `uber/fx`.
- Use `x/settings` for configuration management.

### MCP flags

| Flag | Description | Default |
|------|-------------|---------|
| `--host` | Host or interface for streamable HTTP | `127.0.0.1` |
| `--port` | Streamable HTTP port. `0` selects stdio. | `0` |

The HTTP server has no authentication. Keep it on loopback unless other controls restrict access.

## Interactive mode

Run `ronyup setup` or `ronyup text` without flags to start an interactive TUI:

```bash
ronyup setup
```

For interactive feature creation, set **Feature Parent Directory** for `--featurePrefix`. Set **Group by Template** for `--groupByTemplate`.

Grouping by template places the module under `{featurePrefix}/{template}/{featureDir}/`. The default path is `{featurePrefix}/{featureDir}/`.

## Command reference

```
ronyup
├── setup
│   ├── workspace    Create a new Go workspace
│   ├── feature      Add a feature module to an existing workspace
│   ├── bundle       Create or refresh executable bundles
│   ├── migrate
│   │   └── bundles  Upgrade legacy workspaces to bundle layout
│   └── sync         Refresh scaffold boilerplate in an existing workspace
├── text             Generate translation catalogs
├── lint             Report or fix duplicated x/rkit helpers
├── mcp              Start the MCP server
└── version          Print the ronyup version
```

## Next steps

- [Getting Started](./getting-started.md): Build a server.
- [Cookbook](./cookbook.md): Use service patterns and examples.
- [Architecture](./architecture.md): Learn about internal components.
