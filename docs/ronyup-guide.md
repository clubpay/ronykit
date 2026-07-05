# ronyup: Scaffolding CLI and MCP Server

`ronyup` is the CLI for scaffolding RonyKIT projects and features. It can also run as an MCP server to give AI assistants in your IDE full scaffolding and code generation capabilities.

## Table of Contents

- [Installation](#installation)
- [Create a Workspace](#create-a-workspace)
- [Add a Feature](#add-a-feature)
- [Executable Bundles](#executable-bundles)
- [Upgrade an Older Workspace](#upgrade-an-older-workspace)
- [Feature Templates](#feature-templates)
- [Translation Catalogs](#translation-catalogs)
- [MCP Server](#mcp-server)
  - [Setup for Cursor IDE](#setup-for-cursor-ide)
  - [Setup for JetBrains GoLand](#setup-for-jetbrains-goland)
  - [Available MCP Tools](#available-mcp-tools)
  - [Workflow Example](#workflow-example)
  - [MCP Conventions](#mcp-conventions)
- [Interactive Mode](#interactive-mode)
- [Command Reference](#command-reference)

---

## Installation

With [Homebrew](https://brew.sh/) (macOS/Linux):

```bash
brew install clubpay/tap/ronyup
```

This taps `clubpay/homebrew-tap` and installs a prebuilt `ronyup` binary for your
platform (macOS or Linux, Intel or ARM). On [Homebrew 6+](https://docs.brew.sh/Tap-Trust),
third-party taps require explicit trust; the fully-qualified install above handles that
automatically. Upgrade later with `brew upgrade clubpay/tap/ronyup`. If you tapped
`clubpay/tap` separately and install by short name, run
`brew trust --formula clubpay/tap/ronyup` first.

Prebuilt binaries are also published on [GitHub Releases](https://github.com/clubpay/ronykit/releases)
for tags like `ronyup/vX.Y.Z` (macOS, Linux, and Windows archives).

Or with the Go toolchain:

```bash
go install github.com/clubpay/ronykit/ronyup@latest
```

Verify the installation:

```bash
ronyup --help
```

---

## Create a Workspace

Scaffold a new RonyKIT project with a complete workspace structure:

```bash
ronyup setup workspace \
    --repoDir ./my-api \
    --repoModule github.com/you/my-api
```

This creates:

```
my-api/
├── go.work              # Go workspace file
├── bundles.yaml         # executable bundle manifest
├── Makefile
├── cmd/runner/     # shared bootstrap for all executables
├── cmd/service/         # default all-in-one dev entry point
├── pkg/i18n/            # Translation utilities
├── devops/              # devbox: Helm-managed K8s services
├── docs/                # design docs and guides
└── .git/                # Initialized git repo
```

The workspace is immediately runnable with `make run` or `cd cmd/service && go run .`.

### Workspace kinds

Use `--kind` to choose the layout:

- **`backend`** (default) — a Go-only workspace at `repoDir`, as shown above.
- **`fullstack`** — a `backend/` + `frontend/` split:

```
my-api/
├── backend/          # the Go workspace (go.work, cmd/service, pkg/, feature/, Makefile, .golangci.yml)
├── frontend/         # web/mobile application (framework-agnostic placeholder)
├── devops/           # devbox: Helmfile-managed K8s services (shared)
├── docs/             # design docs and guides (shared)
├── AGENTS.md         # agent guidance (shared)
└── .ai/ .agents/ .cursor/   # AI assistant config (shared)
```

In `fullstack` mode the Go module paths are prefixed with `backend/` (for example `github.com/you/my-api/backend/cmd/service`). Run `ronyup setup feature` and `make`/`go` commands from the `backend/` directory.

```bash
ronyup setup workspace \
    --repoDir ./my-api \
    --repoModule github.com/you/my-api \
    --kind fullstack
```

### Pre-installed agent skills

Every workspace ships with the `ronykit-framework` skill. On top of that, `setup workspace` can pre-install a curated set of general-purpose **agent skills** under `.agents/skills/` — focused on modern code, testing, formatting, debugging, review, and commits.

Use `--skills` (or `-s`) to choose which to install. Pass a comma-separated list of skill IDs, or one of the tokens `default`, `all`, or `none`:

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

Running `ronyup setup` with no flags launches an interactive selector with the kind defaults pre-checked.

| Skill ID               | Category | Default        | Description                                                   |
|------------------------|----------|----------------|---------------------------------------------------------------|
| `go-modern`            | Go       | all kinds      | Idiomatic Go 1.25+/1.26 (generics, iterators, errors, context)|
| `go-testing`           | Go       | all kinds      | Table-driven tests, mandatory x/testkit repo integration + app unit tests |
| `writing-tests`        | Quality  | all kinds      | Language-agnostic TDD discipline and the test pyramid         |
| `code-formatting`      | Quality  | all kinds      | Run formatters/linters; keep diffs clean before finishing     |
| `systematic-debugging` | Quality  | all kinds      | Find root cause before fixing; reproduce, verify, prevent     |
| `code-review`          | Quality  | all kinds      | Prioritized review for correctness, design, and risk          |
| `verification-before-completion` | Quality | all kinds | Require fresh evidence before claiming work is done       |
| `writing-plans`        | Workflow | all kinds      | Plan multi-step work into small, testable tasks first         |
| `conventional-commits` | Workflow | all kinds      | Atomic commits with the Conventional Commits format           |
| `nextjs-modern`        | Frontend | fullstack only | Next.js 16 App Router, Server Components, Server Actions       |
| `frontend-testing`     | Frontend | fullstack only | Vitest + Testing Library and Playwright e2e best practices     |
| `storybook`            | Frontend | fullstack only | Ensure every UI component ships with a Storybook story         |
| `shadcn`               | Frontend | fullstack only | Find, install, compose, and theme shadcn/ui components correctly|
| `frontend-design`      | Frontend | fullstack only | Required before UI bootstrap: design questions, token plan, aesthetics |
| `design-tokens`        | Frontend | fullstack only | Token architecture (primitive→semantic→component), Tailwind/shadcn |

The installed skills are listed in the workspace's `AGENTS.md` with a **skill routing** table (task → which `SKILL.md` files to read).

### Agent enforcement gates

Scaffolded workspaces ship mechanical gates so agents cannot skip design or testing by accident:

| Gate | Enforced by | What it checks |
|------|-------------|----------------|
| Backend SRS/SDD | `scaffold_feature` MCP tool | Approved `docs/design/<feature>-{srs,sdd}.md` before scaffolding |
| Frontend design doc | `frontend/verify.sh` | Approved `docs/design/<app>-frontend-design.md` before UI verify passes |
| Backend repo/app tests | `verify.sh` + Cursor `backend-verify` hook | Integration test per repo port method; unit test per `App` method; tests run green |
| Frontend quality | `frontend/verify.sh` + Cursor `frontend-verify` hook | typecheck, lint, build, test, Storybook stories |

MCP prompt `design-frontend` orchestrates the frontend bootstrap workflow (ask → design doc → approval → init → implement). Backend integration test conventions live in `knowledge://ronyup/architecture/integration-tests`.

> `verification-before-completion` and `writing-plans` are adapted from [obra/superpowers](https://github.com/obra/superpowers) (MIT).

### Workspace flags

| Flag           | Short | Description                                          | Default                |
|----------------|-------|------------------------------------------------------|------------------------|
| `--repoDir`    | `-r`  | Destination directory                                | `./my-repo`            |
| `--repoModule` | `-m`  | Go module path                                       | `github.com/your/repo` |
| `--appName`    | `-a`  | Application name                                     | `myapp`                |
| `--kind`       | `-k`  | Layout: `backend`/`fullstack`                        | `backend`              |
| `--skills`     | `-s`  | Skills to pre-install (IDs, or `default`/`all`/`none`)| kind defaults          |
| `--force`      | `-f`  | Clean destination before setup                       | `false`                |

---

## Add a Feature

Add a new service, job, or gateway module to an existing workspace:

```bash
cd my-api
ronyup setup feature \
    --featureDir users \
    --featureName users \
    --template service \
    --repoModule github.com/you/my-api
```

This creates a fully structured feature module under `feature/users/` with:

- API contracts and handler stubs
- App layer (business logic)
- Repository ports and adapters (with sqlc support)
- Settings/config
- DI wiring (uber/fx)
- Migration setup
- Stub code generator

The feature is automatically added to `go.work` and registered in `cmd/service/features.go`.

By default, features are placed at `{featurePrefix}/{featureDir}/` (for example `feature/users/`). Pass `--groupByTemplate` (or `-g`) to place them at `{featurePrefix}/{template}/{featureDir}/` instead (for example `feature/service/users/`).

### Feature flags

| Flag                | Short | Description                               | Default                |
|---------------------|-------|-------------------------------------------|------------------------|
| `--featurePrefix`   |       | Parent directory for feature modules      | `feature`              |
| `--featureDir`      | `-p`  | Directory name inside the feature prefix  | `my_feature`           |
| `--featureName`     | `-n`  | Feature name                              | `myfeature`            |
| `--template`        | `-t`  | Template: `service`, `job`, or `gateway`  | `service`              |
| `--groupByTemplate` | `-g`  | Group under `{featurePrefix}/{template}/` | `false`                |
| `--repoModule`      | `-m`  | Repository Go module path (auto-detected) | `github.com/your/repo` |
| `--force`           | `-f`  | Replace existing feature directory        | `false`                |

Matching bundles in `bundles.yaml` have their import lists refreshed automatically.

---

## Executable Bundles

Workspaces compile feature modules into one or more executables. The default `cmd/service` binary is an all-in-one dev entry point; production bundles mix and match features at compile time.

```bash
ronyup setup bundle --name auth-api --services feature/auth,feature/session
ronyup setup bundle --gen
cd cmd/service && go run . --service feature/auth
```

See `bundles.yaml` at the Go workspace root. Makefile: `make run`, `make run-bundle BUNDLE=auth-api`, `make build-bundle BUNDLE=auth-api`.

---

## Upgrade an Older Workspace

After upgrading `ronyup`, refresh shared boilerplate and migrate bundle layout once:

```bash
ronyup setup sync
ronyup setup migrate bundles    # from repo root (fullstack) or backend/ — both work
ronyup setup sync --only backend   # optional Makefile bundle targets
```

`setup sync` does not rewrite `cmd/service/main.go`. Use `setup migrate bundles` (preview with `--dry-run`).

---

## Feature Templates

| Template  | Purpose                                                               |
|-----------|-----------------------------------------------------------------------|
| `service` | A standard API service with CRUD endpoints, repo layer, and DI wiring |
| `job`     | A background job or worker module                                     |
| `gateway` | A public-facing API gateway that composes multiple services           |

---

## Translation Catalogs

Generate i18n translation catalogs using `golang.org/x/text/message/pipeline`:

```bash
ronyup text \
    --src-lang en-US \
    --dst-lang en-US,fa-IR,ar-AR \
    --out-dir . \
    --gen-package pkg/i18n \
    --gen-file catalog.go
```

If a `go.work` file is present, the command loads packages from each workspace module.

### Text flags

| Flag               | Short | Description                     | Default             |
|--------------------|-------|---------------------------------|---------------------|
| `--src-lang`       | `-s`  | Source language                 | `en-US`             |
| `--dst-lang`       | `-l`  | Target languages                | `en-US,fa-IR,ar-AR` |
| `--out-dir`        | `-o`  | Output directory                | `.`                 |
| `--gen-package`    | `-g`  | Package path for generated file | —                   |
| `--gen-file`       | `-f`  | Generated file name             | `catalog.go`        |
| `--modules-filter` | `-m`  | Filter go.work modules          | —                   |

---

## MCP Server

`ronyup` includes a built-in [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server that exposes scaffolding and code generation as tools for AI assistants.

This means you can tell your AI assistant things like:

- "Create a new user service with CRUD endpoints and Postgres storage"
- "Add a payment feature with idempotency support"
- "Plan the implementation of a notification service"

And it will use `ronyup` tools to scaffold and implement the code following RonyKit best practices.

### Start the MCP server manually

```bash
ronyup mcp
```

If the server starts and waits for input, it's ready.

### Setup for Cursor IDE

Create or update `.cursor/mcp.json` in your project root:

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

Reopen Cursor (or reload the window). The `ronyup` tools should appear in the AI chat panel.

### Setup for JetBrains GoLand

1. Open **Settings** (Preferences on macOS).
2. Search for **MCP** in the settings search.
3. Add a new **stdio** MCP server:
   - **Name:** `ronyup`
   - **Command:** `ronyup`
   - **Arguments:** `mcp`
4. Save and restart GoLand if tools don't appear immediately.

### Setup for VS Code

Add to your `.vscode/settings.json` or user settings:

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

### Available MCP Tools

| Tool                 | Description                                                              |
|----------------------|-------------------------------------------------------------------------|
| `scaffold_workspace` | Scaffold a new RonyKIT workspace at `path` (`kind`: `backend`/`fullstack`; optional `skills`: IDs or `default`/`all`/`none`) |
| `scaffold_feature`   | Add a `feature/<name>/` module to a workspace                           |

### MCP Prompts

| Prompt                | Description                                 |
|-----------------------|---------------------------------------------|
| `design-new-service`  | Full workflow: SRS → SDD → scaffold → code  |
| `write-srs`           | Write SRS to `docs/design/<feature>-srs.md` |
| `write-sdd`           | Write SDD from approved SRS                 |
| `design-api`          | Design contracts, routes, and handlers      |
| `plan-service`        | Plan a feature (SRS/SDD-first workflow)     |
| `write-service-code`  | Implement service code per SDD              |
| `write-workflow`      | Temporal / `flow` workflows                 |
| `review-architecture` | Review feature layout for compliance        |
| `generate-stubs`      | Stub generation and consumption             |
| `migrate-kit-to-rony` | Migrate from `kit` to `rony` incrementally  |

### Knowledge resources

The server exposes markdown resources (architecture, `x/` packages, characteristics, tool docs). URIs follow `knowledge://ronyup/<category>/<name>`. Use your IDE’s MCP resource browser or see `.agents/skills/ronykit-framework/references/mcp-map.md` in scaffolded workspaces.

### Agent skill

Scaffolded workspaces include the **ronykit-framework** skill under `.agents/skills/` ([Agent Skills](https://agentskills.io/specification) layout). Cursor and other compatible agents discover it automatically. Invoke `/ronykit-framework` in Cursor for workflows that coordinate MCP tools, prompts, and resources (without duplicating the knowledge base).

### Workflow Example

A typical AI-assisted workflow for a **new service feature**:

1. **Requirements (SRS):** MCP prompt `write-srs` or Phase 1 of `design-new-service` → `docs/design/<feature>-srs.md`. Review and approve before continuing.

2. **Design (SDD):** MCP prompt `write-sdd` or Phase 2 of `design-new-service` → `docs/design/<feature>-sdd.md`. Review and approve before scaffolding.

3. **Create the scaffold:** Call `scaffold_feature` (or `scaffold_workspace` for a new repo).

4. **Load knowledge:** Read MCP architecture resources (`design-documents`, `service-structure`, `api-handler-files`, `repo-ports`, etc.) before implementing.

5. **Implement:** MCP prompt `write-service-code`, following the SDD.

6. **Iterate:** Run `make gen-stub` after contract changes; refine handlers and app logic.

### MCP Conventions

The MCP server guides AI assistants to follow these conventions:

- Keep persistence behind `internal/repo/port.go` interfaces with `internal/repo/v0` adapters
- Prefer **Postgres + sqlc** as the default storage stack
- Use a different storage stack only when explicitly requested
- Run `make gen-stub` after contract changes
- Use generated stubs for inter-service calls (not hand-written HTTP clients)
- Wire dependencies through `x/di` and `uber/fx`
- Use `x/settings` for configuration management

### MCP Flags

| Flag             | Description                        | Default    |
|------------------|------------------------------------|------------|
| `--name`         | MCP server name                    | `ronyup`   |
| `--version`      | MCP server version                 | `v0.1.0`   |
| `--instructions` | Custom instructions for AI clients | (built-in) |

---

## Interactive Mode

Running `ronyup setup` or `ronyup text` with no flags starts an interactive TUI that walks you through the configuration:

```bash
ronyup setup
```

When adding a feature interactively, you can set the **Feature Parent Directory** (`--featurePrefix`) and **Group by Template** (`--groupByTemplate`), which places the module under `{featurePrefix}/{template}/{featureDir}/` instead of `{featurePrefix}/{featureDir}/`.

---

## Command Reference

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
├── template         Inspect Go source for template metadata
│   └── generate     (stub — not yet implemented)
├── mcp              Start the MCP server
└── version          Print the ronyup version
```

---

## Next Steps

- [Getting Started](./getting-started.md) — build your first server
- [Cookbook](./cookbook.md) — production patterns and examples
- [Architecture](./architecture.md) — how RonyKit works internally
