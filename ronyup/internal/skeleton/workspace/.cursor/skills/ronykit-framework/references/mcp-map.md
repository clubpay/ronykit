# Ronyup MCP index

Reference for agents using the `ronyup mcp` server. URIs use the form
`knowledge://ronyup/<category>/<name>` unless your client lists resources by name only.

Server instructions (always applied on connect) are embedded in the `ronyup` MCP server
at startup.

## Tools

| Tool | When to use |
|------|-------------|
| `scaffold_workspace` | New Go workspace (`ronyup setup workspace` at `path`) |
| `scaffold_feature` | New `feature/service/<name>/` module in an existing workspace |

Tool docs: `knowledge://ronyup/tools/scaffold_workspace`, `scaffold_feature`.

## Prompts

| Prompt | Purpose |
|--------|---------|
| `design-api` | Contracts, routes, handlers, API documentation |
| `plan-service` | Plan a new service feature before implementation |
| `write-service-code` | Implement handlers, app layer, repos per conventions |
| `write-workflow` | Temporal / `flow` workflows and activities |
| `review-architecture` | Compliance review of an existing feature |
| `generate-stubs` | Generate and consume typed client stubs |
| `migrate-kit-to-rony` | Incremental migration from `kit` to `rony` |

## Architecture resources

Read these when implementing or reviewing service code.

| Resource | Topic |
|----------|--------|
| `workspace-layout` | Repo layout, `cmd/service`, feature registration |
| `service-structure` | `service.go`, `module.go`, `migration.go` |
| `api-handler-files` | `api/service.go`, `api/api_*.go` handlers |
| `domain-layer` | `internal/domain`, errors, types |
| `repo-ports` | Port interfaces and adapters |
| `postgres-sqlc` | Default DB + sqlc layout |
| `module-wiring` | fx module, DB/Redis init |
| `settings-config` | `x/settings` structs and tags |
| `middleware` | Global middleware registration |
| `error-handling` | `rony/errs` patterns |
| `logging` | `logkit` usage |
| `tracing-metrics` | `tracekit`, `meterkit` |
| `flow-workflows` | Workflow module layout |
| `apidoc-generation` | OpenAPI / Postman from descriptors |
| `gen-stub` | Stub generation commands |
| `inter-service-stubs` | Calling other services via stubs |
| `integration-tests` | `x/testkit` patterns |
| `rkit-helpers` | Shared `x/rkit` utilities |
| `migrating-kit-to-rony` | Migration strategy |

## Package resources (`x/` toolkit)

| Resource | Package |
|----------|---------|
| `di` | `x/di` |
| `settings` | `x/settings` |
| `logkit` | `x/telemetry/logkit` |
| `tracekit` | `x/telemetry/tracekit` |
| `meterkit` | `x/telemetry/meterkit` |
| `testkit` | `x/testkit` |
| `i18n` | `x/i18n` |
| `apidoc` | `x/apidoc` |
| `cache` | `x/cache` |
| `flow` | `flow` |
| `rkit` | `x/rkit` |

## Characteristics (keyword routing)

When the user mentions these topics, read the resource before coding.

| Resource | Keywords (partial) |
|----------|-------------------|
| `api` | rest, http, contract |
| `cache` | redis, cache |
| `database` | postgres, mysql |
| `di` | di, dependency inject |
| `i18n` | i18n, locale |
| `idempotent` | idempotent |
| `telemetry` | telemetry, observ |
| `testing` | test, integrat |
| `workflow` | workflow, temporal |

## Typical read order (new service feature)

1. `scaffold_feature` (tool)
2. `architecture/service-structure`
3. `architecture/api-handler-files`
4. `architecture/repo-ports` + `architecture/postgres-sqlc`
5. Prompt `write-service-code`
6. On completion: `architecture/gen-stub` → run `make gen-stub`
