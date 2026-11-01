# ronyup

`ronyup` scaffolds RonyKIT workspaces and features, maintains their boilerplate and agent skills, generates translation catalogs, checks Go code for duplicated `x/rkit` helpers, and exposes the same operations through MCP.

## Install

With Homebrew on macOS or Linux:

```bash
brew install clubpay/tap/ronyup
```

Upgrade with `brew upgrade clubpay/tap/ronyup`. On Homebrew 6+, the fully-qualified name also handles third-party tap trust.

With Go:

```bash
go install github.com/clubpay/ronykit/ronyup@latest
```

From this repository root:

```bash
go install ./ronyup
```

## Commands

- `setup workspace`: create a backend, fullstack, or frontend workspace.
- `setup feature`: add a service feature to an existing Go workspace.
- `setup sync`: refresh scaffold-managed files and installed skills.
- `setup bundle`: create, remove, or regenerate executable bundles.
- `setup migrate bundles`: migrate an older workspace to the bundle layout.
- `text`: extract and generate translation catalogs.
- `lint`: report or fix code that re-implements an `x/rkit` helper.
- `mcp`: run the MCP server over stdio or streamable HTTP.
- `version`: print the version.

Running `ronyup setup` or `ronyup text` without flags opens the interactive UI.

## Create a workspace

```bash
ronyup setup workspace \
  --repoDir ./my-repo \
  --repoModule github.com/your/repo \
  --appName myapp \
  --kind backend
```

Workspace kinds:

- `backend` (default): Go workspace at the repository root.
- `fullstack`: Go workspace under `backend/`; frontend, docs, devops, and AI configuration at the repository root.
- `frontend`: frontend and shared AI configuration without a Go workspace.

Every workspace includes the `ronykit-framework` skill, a `.gitignore` for local configuration and secrets, and verification hooks appropriate to its kind. Select additional bundled skills with `--skills`:

```bash
ronyup setup workspace -r ./my-repo -m github.com/your/repo --skills default
ronyup setup workspace -r ./my-repo -m github.com/your/repo --skills go-modern,go-testing
ronyup setup workspace -r ./my-repo -m github.com/your/repo --skills all
ronyup setup workspace -r ./my-repo -m github.com/your/repo --skills none
```

Skills are installed under `.agents/skills/<id>`. Claude Code uses `.claude/skills/<id>` symlinks to the same files.

`--force` replaces a non-empty destination. Use it only when deleting that destination is intended.

## Add a feature

Run from the Go workspace root or the fullstack repository root:

```bash
ronyup setup feature \
  --featureDir auth \
  --featureName auth \
  --template service
```

The feature is created under `feature/auth/`, added to `go.work`, imported by `cmd/all-in-one/features.go`, and included in matching bundles. Use `--featurePrefix` to change the parent directory or `--groupByTemplate` for `feature/service/auth/`.

Feature paths must stay inside the workspace, and `featureName` must be a valid Go package identifier. The service scaffold includes a teaching CRUD slice, app and repository tests, settings, DI wiring, migrations, and stub-generation commands to replace with the approved design.

The MCP `scaffold_feature` tool additionally requires approved SRS and SDD documents. The direct CLI is a power-user path and does not enforce that document gate.

## Sync an existing workspace

```bash
cd ./my-repo
ronyup setup sync
```

By default, sync only adds missing files. It does not replace or delete existing content. Use `--overwrite` only when existing scaffold-managed content should be replaced.

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

`--skills` accepts `installed` (default), `default`, `all`, `none`, or explicit IDs. Installed mode replaces retired skills with their catalog replacements. Retired directories and legacy Cursor copies are removed only with `--overwrite`; extra user-authored skill files are preserved.

## Executable bundles

```bash
ronyup setup bundle --name auth-api --services feature/auth,feature/session
ronyup setup bundle --gen
ronyup setup bundle --remove auth-api
```

Bundles are declared in `bundles.yaml` and generated under `cmd/<name>/`. The default `all-in-one` bundle imports every feature. Use `--services '*'` for that behavior in another bundle.

Migrate an older workspace before creating bundles:

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

When `go.work` is present, packages are loaded from its `use` entries. Use `--modules-filter` to select specific entries.

## Lint for x/rkit duplicates

```bash
ronyup lint ./...
ronyup lint --fix ./...
```

The linter reports hand-written copies of helpers such as `rkit.Must`, `Coalesce`, `Map`, `Filter`, map collection functions, conversion helpers, and covered standard-library calls. Modules that do not require `x/rkit` are skipped unless `--all-modules` is set.

Exit codes are 0 for no remaining findings, 1 for findings, and 2 when packages cannot be loaded. Silence a deliberate exception with `//rkitcheck:ignore <reason>`.

## MCP server

Start with stdio, the normal IDE configuration:

```bash
ronyup mcp
```

Or streamable HTTP:

```bash
ronyup mcp --host 127.0.0.1 --port 8080
```

HTTP binds to loopback by default. The server has no authentication; do not expose it with `--host 0.0.0.0` outside a trusted environment.

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
| `scaffold_feature` | Add a feature, gated on approved SRS/SDD unless explicitly skipped |
| `sync_workspace` | Refresh selected scaffold sections |
| `sync_skills` | Sync skills and return post-sync structured validation |
| `validate_skills` | Read-only skill content, frontmatter, link, and retirement checks |
| `setup_bundle` | Create, remove, or regenerate executable bundles |
| `migrate_bundles` | Migrate the bundle layout, with optional dry run |
| `check_frontend_design` | Verify an approved, complete frontend design document |
| `check_rkit_usage` | Report or fix duplicated `x/rkit` helpers |

For skill maintenance, call `validate_skills` first. A drift result has `upToDate: false` and per-skill issues; it is not an operational MCP error. Call `sync_skills` with the same selection to repair it. Sync returns structured post-sync validation.

Knowledge resources are available under `knowledge://ronyup/{architecture,packages,characteristics,tools}/…`. Workflow prompts cover service design, API design, frontend design, implementation, stubs, workflows, reviews, and migration.

See `docs/ronyup-guide.md` for the complete guide.
