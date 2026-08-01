# ronyup

`ronyup` is a CLI that scaffolds RonyKIT workspaces and features, and supports i18n catalog generation for Go projects.

## Install

With Homebrew (macOS/Linux):

```bash
brew install clubpay/tap/ronyup
```

On [Homebrew 6+](https://docs.brew.sh/Tap-Trust), third-party taps require explicit trust. The
fully-qualified name above trusts and installs `ronyup` in one step. Upgrade with
`brew upgrade clubpay/tap/ronyup`.

From the repo root:

```bash
go install ./ronyup
```

Or with a module path:

```bash
go install github.com/clubpay/ronykit/ronyup@latest
```

## Commands

- `setup`: create a new workspace, add a feature, migrate bundle layout, sync scaffold files, or manage bundles.
- `text`: extract and generate translation catalogs using `golang.org/x/text/message/pipeline`.
- `mcp`: run `ronyup` as a stdio or streamable-HTTP MCP server for AI assistants.
- `version`: print the `ronyup` version.

Running `ronyup setup` or `ronyup text` with no flags triggers an interactive TUI.

## Usage

### Create a workspace

```bash
ronyup setup workspace \
  --repoDir ./my-repo \
  --repoModule github.com/your/repo
```

This copies the embedded workspace skeleton, initializes a `go.work`, and creates module roots for:

- `pkg/i18n`
- `cmd/all-in-one`

Pass `--kind fullstack` to scaffold a `backend/` + `frontend/` split instead of a Go-only repo. The Go workspace (`go.work`, `cmd/all-in-one`,
`pkg`, `feature`, `Makefile`, `.golangci.yml`) is created under `backend/`, while `devops/`, `docs/`, and the AI assistant config (`.ai/`,
`.agents/`, `.cursor/`, `AGENTS.md`) stay at the root. You can run `setup feature` from the repository root or from `backend/`; feature modules
are always written under the Go workspace root (`backend/` in fullstack mode).

Pass `--kind frontend` to scaffold a frontend-only repo: a `frontend/` application plus shared AI assistant config and `docs/` at the root,
with no Go workspace, `devops/`, or backend verify gate.

### Sync an existing workspace

Refresh scaffold-managed boilerplate after upgrading `ronyup` (or to pick up new skeleton files such as `devops/devbox/`):

```bash
cd ./my-repo
ronyup setup sync
```

By default only **missing** scaffold files are added. Pass `--overwrite` to replace existing scaffold files (for example `AGENTS.md`, hooks,
or skill trees).

```bash
# Add only the new devbox folder without touching existing files
ronyup setup sync --only devops

# Refresh agent guidance and hooks from the latest skeleton
ronyup setup sync --only agents,hooks --overwrite

# Update skills already installed under .agents/skills/
ronyup setup sync --only skills --overwrite --skills installed
```

Sections (`--only`):

| Section    | Updates                                                                                                 |
|------------|---------------------------------------------------------------------------------------------------------|
| `agents`   | `AGENTS.md`                                                                                             |
| `ai`       | `.ai/mcp/`, `.cursor/mcp.json`                                                                          |
| `hooks`    | `.cursor/hooks/`, `hooks.json`                                                                          |
| `devops`   | `devops/devbox/`                                                                                        |
| `docs`     | `docs/design/README.MD`                                                                                 |
| `skills`   | `.agents/skills/ronykit-framework` + installed catalog skills                                           |
| `backend`  | `Makefile`, `verify.sh`, `.golangci.yml`, `bundles.yaml` (when missing; does not migrate `cmd/all-in-one`) |
| `frontend` | `frontend/Makefile`, `verify.sh`, `README.MD`                                                           |
| `all`      | all sections applicable to the detected workspace kind (default)                                        |

Workspace kind is auto-detected from `go.work`, `backend/go.work`, or `frontend/`. In fullstack repos you can run sync from the repository
root or from `backend/`.

### Migrate an older workspace to bundle layout

Workspaces created before executable bundles need a **one-time** migration (sync alone does not rewrite `cmd/all-in-one/main.go`):

```bash
cd ./my-repo   # backend-only repo root, or fullstack repo root (backend/ resolved automatically)
ronyup setup migrate bundles
ronyup setup sync --only backend   # optional: Makefile bundle targets
```

Preview changes with `--dry-run`. The command is idempotent. See [setup migrate bundles](#setup-migrate-bundles) below.

### Add a feature

```bash
ronyup setup feature \
  --featureDir auth \
  --featureName auth \
  --template service
```

The feature is copied from the embedded templates in `internal/skeleton/feature` into `feature/auth/` by default. Use `--featurePrefix` to
change the parent directory, or `--groupByTemplate` (or `-g`) for `feature/service/auth/`.

### Generate translation catalogs

```bash
ronyup text \
  --src-lang en-US \
  --dst-lang en-US,fa-IR,ar-AR \
  --out-dir . \
  --gen-package pkg/i18n \
  --gen-file catalog.go
```

If a `go.work` file is present, the command loads packages from each `use` entry (filtered by `--modules-filter`).

### Run MCP server

```bash
ronyup mcp
```

Starts a stdio MCP server (or streamable HTTP with `--port`). Exposed tools:

- `scaffold_workspace` — initialize a workspace at the given path (delegates to `ronyup setup workspace --repoDir .`)
- `scaffold_feature` — add a feature module (gated on approved SRS/SDD design docs unless `skipDesignGate=true`)

Knowledge resources are served under `knowledge://ronyup/{architecture,packages,characteristics,tools}/…`, and workflow prompts include
`design-new-service`, `write-srs`, `write-sdd`, `design-frontend`, and `migrate-kit-to-rony`.

CLI-only commands (`setup sync`, `setup bundle`, `setup migrate bundles`) are documented as knowledge resources but are not MCP tools.

## Flags

### setup

Global flags:

- `--repoModule`, `-m`: Go module path for the repository.
- `--force`, `-f`: clean destination directory before setup.
- `--custom`, `-c`: custom key/value map for templates.

Workspace flags:

- `--repoDir`, `-r`: destination directory for the workspace.
- `--appName`, `-a`: application name (default `myapp`).
- `--kind`, `-k`: workspace kind, `backend` (default), `fullstack`, or `frontend`.

Feature flags:

- `--featurePrefix`: parent directory for feature modules (default `feature`).
- `--featureDir`, `-p`: destination directory inside the feature prefix.
- `--featureName`, `-n`: feature name.
- `--template`, `-t`: feature template (`service`, `job`, `gateway`).
- `--groupByTemplate`, `-g`: group features under `{featurePrefix}/{template}/`.

### text

- `--src-lang`, `-s`: source language (default `en-US`).
- `--dst-lang`, `-l`: destination languages (default `en-US,fa-IR,ar-AR`).
- `--out-dir`, `-o`: output directory (default `.`).
- `--gen-package`, `-g`: package/path for generated file.
- `--gen-file`, `-f`: generated file name (default `catalog.go`).
- `--modules-filter`, `-m`: go.work module filter list.

### template

- `--templateRepo`, `-r`: custom templates repo (not used yet).
- `--template`, `-t`: template name (placeholder).
- `--custom`, `-c`: custom key/value map for templates.

## Notes

- Only the `service` feature template is currently included under `internal/skeleton/feature`.
- `template generate` is a stub; the `template` command only inspects Go source today.
- `--force`, `--custom`, and `--templateRepo` are accepted but not wired into behavior yet.

# ronyup

`ronyup` is the CLI for RonyKIT scaffolding and tooling. It can:

- Create a new Go workspace with a RonyKIT service layout.
- Add new feature modules (service/job/gateway) to an existing workspace.
- Extract and generate translation catalogs.
- Inspect Go packages for template-related metadata (WIP).

## Installation

```bash
brew install clubpay/tap/ronyup
# or
go install github.com/clubpay/ronykit/ronyup@latest
```

On Homebrew 6+, use the fully-qualified formula name so tap trust is handled automatically.
Upgrade with `brew upgrade clubpay/tap/ronyup`. If you tapped `clubpay/tap` separately and
install by short name, run `brew trust --formula clubpay/tap/ronyup` first. See
[Tap Trust](https://docs.brew.sh/Tap-Trust).

## Usage

```bash
ronyup <command> [subcommand] [flags]
```

Run `ronyup` with no args/flags to enter interactive mode (where supported).

## Commands

### setup

Create a workspace or add a feature module.

```bash
ronyup setup [workspace|feature] [flags]
```

Global flags for `setup`:

```
-m, --repoModule string        go module for the repository (default "github.com/your/repo")
-f, --force                    clean destination directory before setup
-c, --custom stringToString    custom values for the template
```

Notes:

- `--custom` is parsed but not currently applied in the template renderer.
- Running `ronyup setup` with no args/flags starts the interactive prompt.

#### setup workspace

Scaffolds a new workspace and initializes Go workspace/modules.

```bash
ronyup setup workspace -r ./my-repo -m github.com/you/myrepo
```

Flags:

```
-r, --repoDir string    destination directory for the setup (default "./my-repo")
-a, --appName string    application name (default "myapp")
-k, --kind string       workspace kind: backend | fullstack | frontend (default "backend")
```

What it does:

- Copies the embedded workspace skeleton to `repoDir`.
- Creates `go.work`, initializes `pkg/i18n` and `cmd/all-in-one` modules.
- Sets `go` version to `1.25`, runs `go mod tidy` and `go work use .`.
- Initializes a git repo and makes an initial commit.

Kinds:

- `backend` (default): a Go-only workspace at `repoDir`.
- `fullstack`: a `backend/` + `frontend/` split. The Go workspace is created under `backend/` (modules are prefixed with `backend/`, e.g.
  `github.com/you/repo/backend/cmd/all-in-one`), while `devops/`, `docs/`, and AI assistant config (`.ai/`, `.agents/`, `.cursor/`,
  `AGENTS.md`) stay at the root and are shared. Run `setup feature` from the `backend/` directory.
- `frontend`: a frontend-only workspace — a `frontend/` application plus shared AI assistant config (`.ai/`, `.agents/`, `.cursor/`,
  `AGENTS.md`) and `docs/` at the root. No Go workspace, `devops/`, `Makefile`, or backend verify gate is created; only the frontend verify
  stop hook is installed.

#### setup feature

Adds a new feature module under `{featurePrefix}/{featureDir}/` by default (for example `feature/auth/`). Pass `--groupByTemplate` (or `-g`)
to place the module at `{featurePrefix}/{template}/{featureDir}/` instead (for example `feature/service/auth/`).

```bash
ronyup setup feature -m github.com/you/myrepo -p auth -n auth -t service
```

Flags:

```
    --featurePrefix string   prefix for feature folder (default "feature")
-p, --featureDir string      destination directory inside the feature prefix (default "my_feature")
-n, --featureName string     feature name (default "myfeature")
-t, --template string        possible values: service | job | gateway (default "service")
-g, --groupByTemplate        group features by template (default false)
```

Requirements and side effects:

- Must be run from a Go workspace root (a `go.work` file must exist).
- Writes into `{featurePrefix}/{featureDir}/` in the current directory (or `{featurePrefix}/{template}/{featureDir}/` when
  `--groupByTemplate` is set).
- Initializes a Go module for the feature, sets `go` to `1.25`, runs `go mod tidy`, `go fmt`, and adds the module to `go.work`.
- Adds a side-effect import to `cmd/all-in-one/features.go` and runs `go mod tidy` + `go fmt` in `cmd/all-in-one`.
- Use `--force` to replace a non-empty feature directory.

#### setup bundle

Creates or refreshes executable bundles under `cmd/<name>/` that compile in only the selected feature modules. The default `all-in-one` bundle
remains the all-in-one dev binary.

```bash
ronyup setup bundle --name auth-api --services feature/auth,feature/session
ronyup setup bundle --gen
ronyup setup bundle --remove auth-api
```

Flags:

```
-n, --name string             bundle name (cmd/<name>/ directory)
-s, --services strings        feature module paths (settings.ModuleName), or * for all
    --description string      optional description stored in bundles.yaml
    --gen                     regenerate features.go for every bundle from bundles.yaml
    --remove                  remove a bundle and delete cmd/<name>/
```

Requirements and side effects:

- Must be run from a Go workspace root (a `go.work` file must exist).
- Updates `bundles.yaml`, generates `cmd/<name>/main.go` and selective `features.go`, initializes a Go module, and runs `go work use`.
- `ronyup setup feature` also refreshes bundles that include the new feature (or use `"*"`).
- Runtime subsetting within a compiled binary: `go run . --service feature/auth` or `SERVICES=feature/auth,feature/user`.

#### setup migrate bundles

Upgrades workspaces scaffolded before executable bundles to the current layout (`pkg/runner`, `bundles.yaml`, thin `cmd/all-in-one/main.go`).

```bash
ronyup setup migrate bundles
ronyup setup migrate bundles --dry-run
```

Flags:

```
    --dry-run    print planned changes without writing files
```

Requirements and side effects:

- Must be run from a Go workspace root **or** the repository root in fullstack workspaces.
- Idempotent — safe to run multiple times; already-migrated workspaces only refresh bundle `features.go` files.
- Copies `pkg/runner/`, rewrites `cmd/all-in-one/main.go`, removes legacy `middleware.go`/`healthz.go`, `cmd/runner/`, and `internal/runner/` when present,
  creates `bundles.yaml` when missing, runs `go work use ./pkg/runner`.
- Backs up a monolithic main to `cmd/all-in-one/main.go.legacy`.
- Does not modify feature modules under `feature/*`.

### text

Extracts and generates translation catalogs using `golang.org/x/text/message/pipeline`.

```bash
ronyup text [flags]
```

Flags:

```
-s, --src-lang string         source language (default "en-US")
-l, --dst-lang strings        languages to generate (default ["en-US","fa-IR","ar-AR"])
-o, --out-dir string          output path (default ".")
-g, --gen-package string      package or relative path for generated file
-f, --gen-file string         generated file name (default "catalog.go")
-m, --modules-filter strings  modules filter (go.work paths)
```

Behavior:

- If `go.work` exists, it loads all workspace modules (or those in `--modules-filter`).
- Otherwise it scans the current module.
- Prints detected package list before extraction.
- Runs extract → import → merge → export → generate.

Run `ronyup text` with no args/flags to enter the interactive prompt.

### mcp

Run `ronyup` as an MCP server over stdio (default) or streamable HTTP:

```bash
ronyup mcp [--port N]
```

Flags:

```
    --port int   Port for streamable HTTP. If 0 (default), use stdio.
```

Exposed MCP tools:

- `scaffold_workspace`: create a workspace at `path` (`kind`: `backend` | `fullstack` | `frontend`; optional `skills`).
- `scaffold_feature`: create a feature in an existing workspace. Requires approved SRS/SDD under `docs/design/` unless
  `skipDesignGate=true` (only when the user explicitly asks to skip).

Knowledge resources use URIs like `knowledge://ronyup/architecture/service-structure`. Prompts cover design and migration workflows
(e.g. `design-new-service`, `migrate-kit-to-rony`).

## Use `ronyup` MCP in IDEs

Before configuring any IDE, make sure `ronyup` is installed and available on your `PATH`:

```bash
go install github.com/clubpay/ronykit/ronyup@latest
ronyup mcp
```

If the second command starts and waits for input, the MCP server is ready.

### Cursor IDE

Create (or update) `.cursor/mcp.json` in your project:

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

Then reopen Cursor (or reload the window). The `ronyup` MCP tools should be available in Cursor's AI chat/tooling panel.

### JetBrains GoLand

GoLand MCP setup is done from Settings/Preferences:

1. Open `Settings` (`Preferences` on macOS).
2. Search for `MCP` in settings search.
3. Add a new **stdio** MCP server with:
	- Name: `ronyup`
	- Command: `ronyup`
	- Arguments: `mcp`
4. Save settings and restart GoLand if tools do not appear immediately.

After setup, verify from the IDE MCP/AI tools view that server `ronyup` is connected and its tools are listed (`scaffold_workspace`,
`scaffold_feature`).

## Examples

Create a new workspace:

```bash
ronyup setup workspace -r ./my-repo -m github.com/you/myrepo
```

Add a new service feature:

```bash
ronyup setup feature -m github.com/you/myrepo -p auth -n auth -t service
```

Generate translations:

```bash
ronyup text -s en-US -l en-US,fa-IR -o ./i18n -f catalog.go
```

## Requirements

- Go toolchain available on `PATH`.
- Git available on `PATH` for `setup workspace`.
- A Go workspace root (`go.work`) for `setup feature`.
