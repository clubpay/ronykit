---
name: ronykit-framework
description: >-
  Orchestrates RonyKit service development using the ronyup MCP server (knowledge
  resources, prompts, scaffold tools). Use when the user mentions RonyKit, ronyup,
  EdgeServer, contracts, scaffolding a workspace or feature, or implementing API
  handlers and services in RonyKit style.
---

# RonyKit (agent playbook)

Thin orchestration layer. **Domain knowledge lives in the `ronyup` MCP server** — read
resources and prompts there; do not invent layout or conventions from memory.

Full MCP index: [mcp-map.md](mcp-map.md)

## Prerequisites

1. Confirm the `ronyup` MCP server is enabled in the project (`.cursor/mcp.json` or
   `.ai/mcp/mcp.json`: `command` `ronyup`, `args` `["mcp"]`).
2. If MCP is unavailable, ask the user to install `ronyup` and enable MCP — do not
   guess service structure.

## Context

| Workspace | Approach |
|-----------|----------|
| **RonyKit monorepo** (`github.com/clubpay/ronykit`) | Framework work: follow `AGENTS.md` / `CLAUDE.md`, respect `go.work` module boundaries. Do not scaffold into this repo unless testing `ronyup`. |
| **Scaffolded app** (`go.work`, `cmd/service`, `feature/`) | MCP knowledge + tools are the source of truth. |

## Default workflow (app development)

1. **Clarify scope** — MCP prompts `plan-service` or `design-api` when requirements are unclear.
2. **Scaffold** — MCP tools `scaffold_workspace` (new repo) or `scaffold_feature` (existing workspace).
3. **Load knowledge** (read MCP resources before coding):
   - Always: `architecture/service-structure`, `architecture/api-handler-files`
   - Persistence: `architecture/postgres-sqlc`, `architecture/repo-ports`
   - Wiring: `architecture/module-wiring`, `architecture/settings-config`
   - Cross-service: `architecture/inter-service-stubs`, `architecture/gen-stub`
4. **Implement** — MCP prompt `write-service-code`; follow generated files in the feature module.
5. **Characteristics** — If the user mentions caching, i18n, idempotency, workflows, telemetry, etc., read the matching `characteristics/<name>` resource first (see [mcp-map.md](mcp-map.md)).
6. **Finish** — `make gen-stub` in the feature after contract changes; run targeted tests, then workspace `make lint` / `make test` when appropriate.

## Task → MCP routing

| User intent | Start with |
|-------------|------------|
| New repository | `scaffold_workspace` → `architecture/workspace-layout` |
| New service module | `scaffold_feature` → `plan-service` prompt → `write-service-code` |
| API / contract design | `design-api` prompt |
| Architecture review | `review-architecture` prompt |
| Temporal / long-running work | `write-workflow` prompt + `characteristics/workflow` |
| Client stubs | `generate-stubs` prompt + `architecture/gen-stub` |
| Migrate kit → rony | `migrate-kit-to-rony` prompt + `architecture/migrating-kit-to-rony` |

## Hard rules

- Handlers thin; business logic in `internal/app`; persistence behind `internal/repo/port.go`.
- Default storage: Postgres + sqlc in `internal/repo/v0` unless the user requests otherwise.
- Use `x/di`, `x/settings`, `x/telemetry/*`, `rony/errs` — see MCP `packages/*` resources; avoid third-party substitutes.
- Feature Go package name: `<feature>mod` (e.g. `authmod`, not `auth`).
- After contract changes: `make gen-stub` in that feature module.
- Inter-service calls: generated stubs, not hand-written HTTP clients.

## Validation

**Scaffolded app**

```bash
cd feature/service/<feature> && make gen-stub   # after contract changes
cd <module> && go test ./...
make lint && make test                          # from workspace root when feasible
```

**RonyKit monorepo**

```bash
cd <module> && go test ./...
make lint                                       # from repo root when feasible
```

## References (human docs)

- RonyKit docs: https://github.com/clubpay/ronykit/tree/main/docs
- This workspace: `AGENTS.md`
