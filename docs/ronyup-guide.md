# ronyup: Scaffolding CLI and MCP Server

`ronyup` is the CLI for scaffolding RonyKIT projects and features. It can also run as
an MCP server to give AI assistants in your IDE full scaffolding and code generation
capabilities.

## Table of Contents

- [Installation](#installation)
- [Create a Workspace](#create-a-workspace)
- [Add a Feature](#add-a-feature)
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
├── go.work           # Go workspace file
├── Makefile
├── cmd/service/      # Entry point
├── pkg/i18n/         # Translation utilities
└── .git/             # Initialized git repo
```

The workspace is immediately runnable with `cd cmd/service && go run .`.

### Workspace flags

| Flag           | Short | Description                    | Default                |
|----------------|-------|--------------------------------|------------------------|
| `--repoDir`    | `-r`  | Destination directory          | `./my-repo`            |
| `--repoModule` | `-m`  | Go module path                 | `github.com/your/repo` |
| `--force`      | `-f`  | Clean destination before setup | `false`                |

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

This creates a fully structured feature module under `feature/service/users/` with:

- API contracts and handler stubs
- App layer (business logic)
- Repository ports and adapters (with sqlc support)
- Settings/config
- DI wiring (uber/fx)
- Migration setup
- Stub code generator

The feature is automatically added to `go.work` and registered in `cmd/service/features.go`.

### Feature flags

| Flag            | Short | Description                              | Default                |
|-----------------|-------|------------------------------------------|------------------------|
| `--featureDir`  | `-p`  | Directory name inside the repo           | `my_feature`           |
| `--featureName` | `-n`  | Feature name                             | `myfeature`            |
| `--template`    | `-t`  | Template: `service`, `job`, or `gateway` | `service`              |
| `--repoModule`  | `-m`  | Repository Go module path                | `github.com/your/repo` |
| `--force`       | `-f`  | Replace existing feature directory       | `false`                |

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

`ronyup` includes a built-in [Model Context Protocol (MCP)](https://modelcontextprotocol.io/)
server that exposes scaffolding and code generation as tools for AI assistants.

This means you can tell your AI assistant things like:

- "Create a new user service with CRUD endpoints and Postgres storage"
- "Add a payment feature with idempotency support"
- "Plan the implementation of a notification service"

And it will use `ronyup` tools to scaffold and implement the code following RonyKit
best practices.

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

Reopen Cursor (or reload the window). The `ronyup` tools should appear in the AI
chat panel.

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

| Tool                | Description                                                    |
|---------------------|----------------------------------------------------------------|
| `list_templates`    | List embedded scaffold template files                          |
| `read_template`     | Read the contents of a specific template file                  |
| `create_workspace`  | Scaffold a new RonyKIT workspace                               |
| `create_feature`    | Add a feature module to an existing workspace                  |
| `plan_service`      | Generate a dry-run implementation plan with architecture hints |
| `implement_service` | Write starter implementation code for a planned service        |

### Workflow Example

A typical AI-assisted workflow:

1. **Plan the service:** Ask your AI to plan a new feature. It will call `plan_service`
   to generate an architecture plan with recommended packages and file layout.

2. **Create the scaffold:** The AI calls `create_feature` to generate the skeleton code.

3. **Implement the service:** The AI calls `implement_service` to write the base
   implementation — contracts, app logic, and repo ports.

4. **Iterate:** You refine the generated code, add business logic, and the AI
   assists with handler implementation following the established patterns.

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

Running `ronyup setup` or `ronyup text` with no flags starts an interactive TUI
that walks you through the configuration:

```bash
ronyup setup
```

---

## Command Reference

```
ronyup
├── setup
│   ├── workspace    Create a new Go workspace
│   └── feature      Add a feature module to an existing workspace
├── text             Generate translation catalogs
├── template         Inspect Go source for template metadata
│   └── generate     (stub — not yet implemented)
└── mcp              Start the MCP server
```

---

## Next Steps

- [Getting Started](./getting-started.md) — build your first server
- [Cookbook](./cookbook.md) — production patterns and examples
- [Architecture](./architecture.md) — how RonyKit works internally
