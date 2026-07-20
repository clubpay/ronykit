---

## name: ronykit-framework description: >- Orchestrates RonyKit service development using the ronyup MCP server (knowledge resources, prompts, scaffold tools). Use when the user mentions RonyKit, ronyup, EdgeServer, contracts, scaffolding a workspace or feature, implementing API handlers and services, frontend bootstrap, integration tests, or design documents in RonyKit style.

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
| **Scaffolded app** (`go.work`, `cmd/all-in-one`, `feature/`) | MCP knowledge + tools are the source of truth. Fullstack scaffolds nest the Go workspace under `backend/` (run `go`/`make`/`ronyup setup feature` there); `docs/`, `devops/` and AI config stay at the repo root. See `architecture/workspace-layout`. |

## Skill routing (read SKILL.md — required)

Do not rely on memory or optional auto-discovery. **Open and read** `.agents/skills/<id>/SKILL.md` for every row that applies:

| Task | Read (in order) |
|------|-----------------|
| New backend feature | this skill → MCP `design-new-service` → `clean-architecture` |
| Repo / persistence | `go-testing` → MCP `architecture/integration-tests` |
| Layer / boundary review | `clean-architecture`, `code-review` |
| Refactoring (tests exist) | `refactoring-patterns`, `verification-before-completion` |
| Untested / legacy code | `working-with-legacy-code` → `go-testing` → `refactoring-patterns` |
| Production hardening | `release-it` → MCP `characteristics/telemetry` |
| Bootstrap frontend | `frontend-design`, `design-tokens`, `typography` → MCP `design-frontend` |
| Dashboard / admin UI | `dashboard-ui`, `shadcn`, `ux-quality` |
| Agent-friendly UI / WebMCP | `webmcp`, `nextjs-modern`, `ux-quality` |
| Next.js / routes | `nextjs-modern`, `shadcn` |
| UI components | `shadcn`, `storybook`, `ux-quality` |
| Frontend tests | `frontend-testing`, `verification-before-completion` |
| Before claiming done | `verification-before-completion`, `code-formatting` |

## Default workflow (app development)

1. **Clarify scope** — For new services, use `design-new-service` (SRS → SDD → scaffold → code) or `write-srs` then `write-sdd` when doing phases separately. Use `design-api` for API-only design. For new frontends, use `design-frontend` (design doc → approval → init stack → implement).
2. **Documents** — Backend: SRS to `docs/design/<feature>-srs.md`, SDD to `docs/design/<feature>-sdd.md`. Frontend: `docs/design/<app>-frontend-design.md`. Get user approval before scaffolding or UI code. Read `architecture/design-documents` and `architecture/frontend-design-documents`.
3. **Scaffold** — MCP tools `scaffold_workspace` (new repo) or `scaffold_feature` (existing workspace).
4. **Load knowledge** (read MCP resources before coding):
- Always: `architecture/service-structure`, `architecture/api-handler-files`
- Persistence: `architecture/postgres-sqlc`, `architecture/repo-ports`, `architecture/integration-tests` (+ `architecture/table-partitioning` when data is expected to grow)
- Wiring: `architecture/module-wiring`, `architecture/settings-config`
- Cross-service: `architecture/inter-service-stubs`, `architecture/gen-stub`
5. **Implement** — MCP prompt `write-service-code` (SDD is source of truth); follow generated files in the feature module.
6. **Characteristics** — If the user mentions caching, i18n, idempotency, workflows, telemetry, etc., read the matching `characteristics/<name>` resource first (see [references/mcp-map.md](references/mcp-map.md)).
7. **Finish** — `make gen-stub` in the feature after contract changes; run `make verify` (backend) and `frontend/verify.sh` (fullstack UI); then workspace `make lint` / `make test` when appropriate.

## Task → MCP routing

| User intent                  | Start with                                                                                    |
|------------------------------|-----------------------------------------------------------------------------------------------|
| New repository               | `scaffold_workspace` → `architecture/workspace-layout`                                        |
| New service module           | `design-new-service` or `write-srs` → `write-sdd` → `scaffold_feature` → `write-service-code` |
| Bootstrap frontend           | `design-frontend` → `architecture/frontend-design-documents`                                  |
| API / contract design        | `design-api` prompt                                                                           |
| Session / passthrough proxy  | `architecture/handler-relay` + `rony.WithRelay` (not `WithUnary`)                             |
| Architecture review          | `review-architecture` prompt                                                                  |
| Temporal / long-running work | `write-workflow` prompt + `characteristics/workflow`                                          |
| Client stubs                 | `generate-stubs` prompt + `architecture/gen-stub`                                             |
| Migrate kit → rony           | `migrate-kit-to-rony` prompt + `architecture/migrating-kit-to-rony`                           |
| Upgrade old workspace layout | `migrate_bundles` tool doc → `ronyup setup migrate bundles`                                   |

## Hard rules

- **Document-first, always.** Every new/changed backend feature requires an approved SRS + SDD (`docs/design/<feature>-srs.md`, `…-sdd.md`) before scaffolding or coding — even for quick edits or "just write the code" requests. Write with `status: draft`; only the user sets `status: approved`. `scaffold_feature` enforces this gate. Read `architecture/design-documents`.
- **Frontend design is document-first.** Before initializing a frontend stack or writing UI, ask aesthetic/design questions and write `docs/design/<app>-frontend-design.md` with token plan and design-system rules; get user approval. Read skills `frontend-design`, `design-tokens`, `typography` and MCP `design-frontend`. `frontend/verify.sh` enforces the approved design doc gate.
- **Clarify frontend topology before any frontend work.** Never assume a single frontend app. Before creating or editing anything under `frontend/`, ask whether there is one app or multiple; for multiple, each app lives in `frontend/<app-name>/` — confirm which app (and stack) this change targets before proceeding.
- **Repo integration tests are mandatory.** Every repository port method needs an `x/testkit` integration test in `internal/repo/integration_test/` (happy path, not-found, conflict) that you **run** and confirm passes. Read `architecture/integration-tests` and skill `go-testing`.
- **App unit tests are mandatory.** Every exported `App` method in `internal/app/` needs a unit test.
- Handlers thin; business logic in `internal/app`; persistence behind `internal/repo/port.go`.
- Default storage: Postgres + sqlc in `internal/repo/v0` unless the user requests otherwise.
- **Package selection is mandatory.** Before hand-rolling a helper or importing a stdlib/third-party package, use the RonyKIT equivalent. Read `architecture/package-selection` (the full reach-for-X → use-Y map) and the relevant `packages/*` resource. Use `x/rkit` (IDs, JSON/byte casts, string↔number, case, collections), `x/di`, `x/settings`, `x/telemetry/*`, `x/datasource`, `x/cache`, `x/ratelimit`, `x/batch`, `x/p`, `x/i18n`, `x/apidoc`, and `rony/errs` — avoid third-party/stdlib substitutes.
- **Workflows: `flow` only.** Never import `go.temporal.io/sdk` directly — it's denied by the workspace `.golangci.yml`.
- Feature Go package name: `<feature>mod` (e.g. `authmod`, not `auth`).
- After contract changes: `make gen-stub` in that feature module.
- Inter-service calls: generated stubs, not hand-written HTTP clients.
- **Passthrough HTTP/WebSocket proxy routes** use `rony.WithRelay` + `RelayCtx.Relay()` only — never `WithUnary`/`WithRawUnary` or `kit.RawMessage` hacks. Read `architecture/handler-relay`. Static gateway proxy: `rony.WithReverseProxy`.
- `make lint` failures from `depguard` are design violations (forbidden imports), not formatting nits — fix by switching to the RonyKIT package.

## Validation

**Scaffolded app**

```bash
cd feature/<feature> && make gen-stub   # after contract changes
make verify                             # backend: repo integration + app unit tests
bash frontend/verify.sh                 # fullstack: design doc + lint/build/test/stories
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
