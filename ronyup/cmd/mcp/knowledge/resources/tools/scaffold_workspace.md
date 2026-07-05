---
name: scaffold_workspace
---

Initialize a new RonyKIT workspace at the given directory by delegating to `ronyup setup workspace`.

## Extended Guidance

The tool runs `ronyup setup workspace` with the provided `path` as the working directory. The `kind` argument selects the layout:

- **`backend`** (default) — a Go-only workspace at `path`, containing `go.work`, `bundles.yaml`, `cmd/runner/`, `cmd/service/`, `pkg/i18n/`, an empty `feature/` tree, `devops/` (optional `devops/devbox/` Kubernetes cluster), `docs/`, and a `.ai/mcp/mcp.json` for IDE integration.
- **`fullstack`** — a `backend/` + `frontend/` split. The Go workspace (`go.work`, `bundles.yaml`, `cmd/runner/`, `cmd/service/`, `pkg/`, `feature/`, `Makefile`, `.golangci.yml`) is created under `backend/`, while `devops/`, `docs/`, and the AI assistant config (`.ai/`, `.agents/`, `.cursor/`, `AGENTS.md`) stay at the root and are shared. A framework-agnostic `frontend/` placeholder is created for the web/mobile app.
- **`frontend`** — a frontend-only workspace: a framework-agnostic `frontend/` placeholder plus shared AI assistant config (`.ai/`, `.agents/`, `.cursor/`, `AGENTS.md`) and `docs/` at the root. No Go workspace, `devops/`, `Makefile`, or backend verify gate is created; only the frontend verify stop hook is installed.

For `fullstack`, Go module paths are prefixed with `backend/` (e.g. `<repoModule>/backend/cmd/service`). Run `scaffold_feature` (and `go`/`make` commands) against the `backend/` directory; the design gate still finds `docs/design` at the repository root.

After scaffolding, add feature modules with the `scaffold_feature` tool, then run `make tidy && make lint && make test` from the workspace root (the `backend/` directory in `fullstack` mode).

For workspaces created with an older `ronyup`, run `ronyup setup migrate bundles` once (see `knowledge://ronyup/tools/migrate_bundles`).
