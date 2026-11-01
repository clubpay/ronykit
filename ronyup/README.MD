# ronyup

`ronyup` scaffolds RonyKIT workspaces and features. It maintains boilerplate and agent skills, generates translation catalogs, and checks for duplicated `x/rkit` helpers. The MCP server exposes the same operations.

## Install

Install with Homebrew on macOS or Linux:

```bash
brew install clubpay/tap/ronyup
```

Upgrade with `brew upgrade clubpay/tap/ronyup`. On Homebrew 6 and later, the fully qualified name also handles third-party tap trust.

Install with Go:

```bash
go install github.com/clubpay/ronykit/ronyup@latest
```

Install from this repository root:

```bash
go install ./ronyup
```

## Commands

- `setup workspace`: Create a backend, fullstack, or frontend workspace.
- `setup feature`: Add a service feature to an existing Go workspace.
- `setup sync`: Refresh scaffold-managed files and installed skills.
- `setup bundle`: Create, remove, or regenerate executable bundles.
- `setup migrate bundles`: Migrate an older workspace to the bundle layout.
- `text`: Extract and generate translation catalogs.
- `lint`: Report or fix code that implements an `x/rkit` helper again.
- `mcp`: Run the MCP server over stdio or streamable HTTP.
- `version`: Print the version.

Run `ronyup setup` or `ronyup text` without flags to open the interactive UI.

## Create a workspace

```bash
ronyup setup workspace \
  --repoDir ./my-repo \
  --repoModule github.com/your/repo \
  --appName myapp \
  --kind backend
```

Workspace kinds:

- `backend` is the default. It creates a Go workspace at the repository root.
- `fullstack` creates the Go workspace under `backend/`. Frontend, docs, devops, and AI configuration stay at the repository root.
- `frontend` creates frontend and shared AI configuration without a Go workspace.

Every workspace includes the `ronykit-framework` skill. It also includes a `.gitignore` for local configuration and secrets. Verification hooks match the workspace kind.

Select additional bundled skills with `--skills`:

```bash
ronyup setup workspace -r ./my-repo -m github.com/your/repo --skills default
ronyup setup workspace -r ./my-repo -m github.com/your/repo --skills go-modern,go-testing
ronyup setup workspace -r ./my-repo -m github.com/your/repo --skills all
ronyup setup workspace -r ./my-repo -m github.com/your/repo --skills none
```

Skills are installed under `.agents/skills/<id>`. Claude Code uses `.claude/skills/<id>` symlinks to the same files.

`--force` replaces a non-empty destination. Use it only when you intend to delete that destination.

## Add a feature

Run this command from the Go workspace root or the fullstack repository root:

```bash
ronyup setup feature \
  --featureDir auth \
  --featureName auth \
  --template service
```

The command creates the feature under `feature/auth/`. It adds the feature to `go.work` and `cmd/all-in-one/features.go`. Matching bundles also include the feature.

Use `--featurePrefix` to change the parent directory. Use `--groupByTemplate` for `feature/service/auth/`.

Feature paths must stay inside the workspace. `featureName` must be a valid Go package identifier.

The service scaffold includes a teaching CRUD slice, app and repository tests, settings, dependency injection, migrations, and stub-generation commands. Replace the teaching code with the approved design.

The MCP `scaffold_feature` tool also requires approved SRS and SDD documents. The direct CLI is for advanced users and does not enforce this document gate.

## Sync an existing workspace

```bash
cd ./my-repo
ronyup setup sync
```

By default, sync adds only missing files. It does not replace or delete existing content. Use `--overwrite` only when you intend to replace scaffold-managed content.

```bash
# Add new devbox files
ronyup setup sync --only devops

# Refresh agent guidance and hooks
ronyup setup sync --only agents,hooks --overwrite

# Update installed skills
ronyup setup sync --only skills --skills installed --overwrite

# Install the current default skill set
ronyup setup sync --only skills --skills default --overwrite
```

Sections for `--only`:

| Section | Managed paths |
|---------|---------------|
| `agents` | `AGENTS.md`, `CLAUDE.md`, `.gitignore`, editor and agent settings |
| `ai` | `.ai/mcp/`, `.cursor/mcp.json` |
| `hooks` | Cursor and Claude verification hooks |
| `devops` | `devops/devbox/` |
| `docs` | `docs/design/README.MD` |
| `skills` | `.agents/skills/` and corresponding Claude links |
| `backend` | Backend Makefile, verify script, lint config, bundle manifest, feature README |
| `frontend` | Frontend Makefile, verify script, README |
| `all` | Every section applicable to the detected workspace kind |

`--skills` accepts `installed`, `default`, `all`, `none`, or explicit IDs. The default is `installed`.

Installed mode replaces retired skills with their catalog replacements. Sync removes retired directories and legacy Cursor copies only with `--overwrite`. It preserves user-authored skill files.

## Executable bundles

```bash
ronyup setup bundle --name auth-api --services feature/auth,feature/session
ronyup setup bundle --gen
ronyup setup bundle --remove auth-api
```

Declare bundles in `bundles.yaml`. The command generates them under `cmd/<name>/`. The default `all-in-one` bundle imports every feature. Use `--services '*'` for the same behavior in another bundle.

Migrate an older workspace before you create bundles:

```bash
ronyup setup migrate bundles --dry-run
ronyup setup migrate bundles
```

Migration is idempotent. Sync alone does not rewrite `cmd/all-in-one/main.go`.

## Translation catalogs

```bash
ronyup text \
  --src-lang en-US \
  --dst-lang en-US,fa-IR,ar-AR \
  --out-dir . \
  --gen-package pkg/i18n \
  --gen-file catalog.go
```

If `go.work` exists, the command loads packages from its `use` entries. Use `--modules-filter` to select specific entries.

## Lint for x/rkit duplicates

```bash
ronyup lint ./...
ronyup lint --fix ./...
```

The linter reports manual copies of `rkit.Must`, `Coalesce`, `Map`, `Filter`, map collection functions, and conversion helpers. It also reports covered standard-library calls.

The linter skips modules that do not require `x/rkit`. Set `--all-modules` to include them.

Exit code 0 means no remaining findings. Exit code 1 means findings remain. Exit code 2 means packages cannot load.

For a deliberate exception, add `//rkitcheck:ignore <reason>`.

## MCP server

Start with stdio for the standard IDE configuration:

```bash
ronyup mcp
```

Or start streamable HTTP:

```bash
ronyup mcp --host 127.0.0.1 --port 8080
```

HTTP binds to loopback by default. The server has no authentication. Do not expose it with `--host 0.0.0.0` outside a trusted environment.

IDE configuration:

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

Available tools:

| Tool | Purpose |
|------|---------|
| `scaffold_workspace` | Create a workspace at `path` |
| `scaffold_feature` | Add a feature. Approved SRS and SDD documents are required unless explicitly skipped. |
| `sync_workspace` | Refresh selected scaffold sections |
| `sync_skills` | Sync skills and return structured validation results |
| `validate_skills` | Check skill content, frontmatter, links, and retirement without changes |
| `setup_bundle` | Create, remove, or regenerate executable bundles |
| `migrate_bundles` | Migrate the bundle layout with an optional dry run |
| `check_frontend_design` | Verify an approved and complete frontend design document |
| `check_rkit_usage` | Report or fix duplicated `x/rkit` helpers |

For skill maintenance, call `validate_skills` first. A drift result has `upToDate: false` and issues for each skill. This result is not an operational MCP error.

Call `sync_skills` with the same selection to repair drift. Sync returns structured validation results.

Knowledge resources use `knowledge://ronyup/{architecture,packages,characteristics,tools}/…`. Workflow prompts cover service design, API design, frontend design, implementation, stubs, workflows, reviews, and migration.

See `docs/ronyup-guide.md` for the complete guide.
