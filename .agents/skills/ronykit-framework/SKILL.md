---

## name: ronykit-framework description: >- Orchestrates RonyKit service development using the ronyup MCP server (knowledge resources, prompts, scaffold tools). Use when the user mentions RonyKit, ronyup, EdgeServer, contracts, scaffolding a workspace or feature, or implementing API handlers and services in RonyKit style.

# RonyKit (agent playbook)

Thin orchestration layer. **Domain knowledge lives in the `ronyup` MCP server** — read resources and prompts there; do not invent layout or conventions from memory.

Full MCP index: [references/mcp-map.md](references/mcp-map.md)

## Prerequisites

1. Confirm the `ronyup` MCP server is enabled in the project (`.cursor/mcp.json` or `.ai/mcp/mcp.json`: `command` `ronyup`, `args` `["mcp"]`).
2. If MCP is unavailable, ask the user to install `ronyup` and enable MCP — do not guess service structure.

## Context

| Workspace                                                 | Approach                                                                                                                                       |
|-----------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------|
| **RonyKit monorepo** (`github.com/clubpay/ronykit`)       | Framework work: follow `AGENTS.md` / `CLAUDE.md`, respect `go.work` module boundaries. Do not scaffold into this repo unless testing `ronyup`. |
| **Scaffolded app** (`go.work`, `cmd/service`, `feature/`) | MCP knowledge + tools are the source of truth. Fullstack scaffolds nest the Go workspace under `backend/` (run `go`/`make`/`ronyup setup feature` there); `docs/`, `devops/` and AI config stay at the repo root. See `architecture/workspace-layout`. |

## Default workflow (app development)

1. **Clarify scope** — For new services, use `design-new-service` (SRS → SDD → scaffold → code) or `write-srs` then `write-sdd` when doing phases separately. Use `design-api` for API-only design.
2. **Documents** — Write SRS to `docs/design/<feature>-srs.md`, SDD to `docs/design/<feature>-sdd.md`; get user approval before scaffolding. Read `architecture/design-documents`, `srs-template`, `sdd-template`.
3. **Scaffold** — MCP tools `scaffold_workspace` (new repo) or `scaffold_feature` (existing workspace).
4. **Load knowledge** (read MCP resources before coding):
- Always: `architecture/service-structure`, `architecture/api-handler-files`
- Persistence: `architecture/postgres-sqlc`, `architecture/repo-ports`
- Wiring: `architecture/module-wiring`, `architecture/settings-config`
- Cross-service: `architecture/inter-service-stubs`, `architecture/gen-stub`
5. **Implement** — MCP prompt `write-service-code` (SDD is source of truth); follow generated files in the feature module.
6. **Characteristics** — If the user mentions caching, i18n, idempotency, workflows, telemetry, etc., read the matching `characteristics/<name>` resource first (see [references/mcp-map.md](references/mcp-map.md)).
7. **Finish** — `make gen-stub` in the feature after contract changes; run targeted tests, then workspace `make lint` / `make test` when appropriate.

## Task → MCP routing

| User intent                  | Start with                                                                                    |
|------------------------------|-----------------------------------------------------------------------------------------------|
| New repository               | `scaffold_workspace` → `architecture/workspace-layout`                                        |
| New service module           | `design-new-service` or `write-srs` → `write-sdd` → `scaffold_feature` → `write-service-code` |
| API / contract design        | `design-api` prompt                                                                           |
| Architecture review          | `review-architecture` prompt                                                                  |
| Temporal / long-running work | `write-workflow` prompt + `characteristics/workflow`                                          |
| Client stubs                 | `generate-stubs` prompt + `architecture/gen-stub`                                             |
| Migrate kit → rony           | `migrate-kit-to-rony` prompt + `architecture/migrating-kit-to-rony`                           |

## Hard rules

- **Document-first, always.** Every new/changed backend feature requires an approved SRS + SDD (`docs/design/<feature>-srs.md`, `…-sdd.md`) before scaffolding or coding — even for quick edits or "just write the code" requests. Write with `status: draft`; only the user sets `status: approved`. `scaffold_feature` enforces this gate. Read `architecture/design-documents`.
- **Clarify frontend topology before any frontend work.** Never assume a single frontend app. Before creating or editing anything under `frontend/`, ask whether there is one app or multiple; for multiple, each app lives in `frontend/<app-name>/` — confirm which app (and stack) this change targets before proceeding.
- Handlers thin; business logic in `internal/app`; persistence behind `internal/repo/port.go`.
- Default storage: Postgres + sqlc in `internal/repo/v0` unless the user requests otherwise.
- **Package selection is mandatory.** Before hand-rolling a helper or importing a stdlib/third-party package, use the RonyKIT equivalent. Read `architecture/package-selection` (the full reach-for-X → use-Y map) and the relevant `packages/*` resource. Use `x/rkit` (IDs, JSON/byte casts, string↔number, case, collections), `x/di`, `x/settings`, `x/telemetry/*`, `x/datasource`, `x/cache`, `x/ratelimit`, `x/batch`, `x/p`, `x/i18n`, `x/apidoc`, and `rony/errs` — avoid third-party/stdlib substitutes.
- **Workflows: `flow` only.** Never import `go.temporal.io/sdk` directly — it's denied by the workspace `.golangci.yml`.
- Feature Go package name: `<feature>mod` (e.g. `authmod`, not `auth`).
- After contract changes: `make gen-stub` in that feature module.
- Inter-service calls: generated stubs, not hand-written HTTP clients.
- `make lint` failures from `depguard` are design violations (forbidden imports), not formatting nits — fix by switching to the RonyKIT package.

## Validation

**Scaffolded app**

```bash
cd feature/<feature> && make gen-stub   # after contract changes
cd <module> && go test ./...
make lint && make test                          # from workspace root when feasible
```

**RonyKit monorepo**

```bash
cd <module> && go test ./...
make lint                                       # from repo root when feasible
```

## References (human docs)

- `docs/ronyup-guide.md` — CLI and MCP setup
- `docs/architecture.md`, `docs/getting-started.md`
- Monorepo: `AGENTS.md`, `CLAUDE.md`
