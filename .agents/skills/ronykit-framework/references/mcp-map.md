# Ronyup MCP index

Reference for agents using the `ronyup mcp` server. URIs use the form `knowledge://ronyup/<category>/<name>` unless your client lists
resources by name only.

Server instructions (always applied on connect) are embedded from `ronyup/cmd/mcp/knowledge/server/instructions.md`.

## Tools

| Tool                 | When to use                                                                       |
|----------------------|-----------------------------------------------------------------------------------|
| `scaffold_workspace` | New Go workspace (`ronyup setup workspace` at `path`)                             |
| `scaffold_feature`   | New `feature/<name>/` module (`groupByTemplate` for `feature/<template>/<name>/`) |

CLI-only (read `knowledge://ronyup/tools/*` docs): `sync_workspace`, `setup_bundle`, `migrate_bundles`.

Tool docs: `knowledge://ronyup/tools/scaffold_workspace`, `scaffold_feature`, `sync_workspace`, `setup_bundle`, `migrate_bundles`.

## Prompts

| Prompt                | Purpose                                           |
|-----------------------|---------------------------------------------------|
| `design-new-service`  | Full workflow: SRS → SDD → scaffold → implement   |
| `design-frontend`     | Frontend bootstrap: design doc → approval → init → implement |
| `write-srs`           | Write SRS to `docs/design/<feature>-srs.md`       |
| `write-sdd`           | Write SDD from approved SRS                       |
| `design-api`          | Contracts, routes, handlers, API documentation    |
| `plan-service`        | Plan a feature (points to SRS/SDD-first workflow) |
| `write-service-code`  | Implement handlers, app layer, repos per SDD      |
| `write-workflow`      | Temporal / `flow` workflows and activities        |
| `review-architecture` | Compliance review of an existing feature          |
| `generate-stubs`      | Generate and consume typed client stubs           |
| `migrate-kit-to-rony` | Incremental migration from `kit` to `rony`        |

## Architecture resources

Read these when implementing or reviewing service code.

| Resource                | Topic                                            |
|-------------------------|--------------------------------------------------|
| `package-selection`     | Mandatory reach-for-X → use-Y package map        |
| `design-documents`      | SRS/SDD workflow, paths, gate rules              |
| `frontend-design-documents` | Frontend design doc workflow, paths, gate rules |
| `frontend-design-template`  | Frontend design doc outline (tokens, rules)     |
| `srs-template`          | SRS section outline (IEEE 830)                   |
| `sdd-template`          | SDD section outline mapped to RonyKIT modules    |
| `workspace-layout`      | Repo layout, bundles, `cmd/all-in-one`, feature registration |
| `workspace-commands`    | Where to run `ronyup setup` commands (go root vs repo root) |
| `executable-bundles`    | Compile-time bundles, runtime `--service`, Makefile targets |
| `service-structure`     | `service.go`, `module.go`, `migration.go`        |
| `api-handler-files`     | `api/service.go`, `api/api_*.go` handlers        |
| `domain-layer`          | `internal/domain`, errors, types                 |
| `repo-ports`            | Port interfaces and adapters                     |
| `postgres-sqlc`         | Default DB + sqlc layout                         |
| `table-partitioning`    | Growing tables: strategy, time partitions, automated maintenance |
| `module-wiring`         | fx module, DB/Redis init                         |
| `settings-config`       | `x/settings` structs and tags                    |
| `middleware`            | Global middleware registration                   |
| `handler-relay`         | Dynamic HTTP/WebSocket relay (`WithRelay`)       |
| `error-handling`        | `rony/errs` patterns                             |
| `logging`               | `logkit` usage                                   |
| `tracing-metrics`       | `tracekit`, `meterkit`                           |
| `flow-workflows`        | Workflow module layout                           |
| `apidoc-generation`     | OpenAPI / Postman from descriptors               |
| `gen-stub`              | Stub generation commands                         |
| `inter-service-stubs`   | Calling other services via stubs                 |
| `integration-tests`     | `x/testkit` patterns                             |
| `rkit-helpers`          | Shared `x/rkit` utilities                        |
| `migrating-kit-to-rony` | Migration strategy                               |

## Package resources (`x/` toolkit)

| Resource     | Package                |
|--------------|------------------------|
| `di`         | `x/di`                 |
| `settings`   | `x/settings`           |
| `logkit`     | `x/telemetry/logkit`   |
| `tracekit`   | `x/telemetry/tracekit` |
| `meterkit`   | `x/telemetry/meterkit` |
| `testkit`    | `x/testkit`            |
| `i18n`       | `x/i18n`               |
| `apidoc`     | `x/apidoc`             |
| `cache`      | `x/cache`              |
| `datasource` | `x/datasource`         |
| `ratelimit`  | `x/ratelimit`          |
| `batch`      | `x/batch`              |
| `p`          | `x/p`                  |
| `flow`       | `flow`                 |
| `rkit`       | `x/rkit`               |

## Characteristics (keyword routing)

When the user mentions these topics, read the resource before coding.

| Resource     | Keywords (partial)    |
|--------------|-----------------------|
| `api`        | rest, http, api, relay, proxy |
| `cache`      | redis, cache          |
| `database`   | postgres, mysql, partition, retention, growth |
| `di`         | di, dependency inject |
| `i18n`       | i18n, locale          |
| `idempotent` | idempotent            |
| `telemetry`  | telemetry, observ     |
| `testing`    | test, integrat        |
| `workflow`   | workflow, temporal    |

## Typical read order (new service feature)

1. `design-new-service` or `write-srs` → `write-sdd` (prompts)
2. `architecture/design-documents`, `srs-template`, `sdd-template`
3. `scaffold_feature` (tool)
4. `architecture/service-structure`
5. `architecture/api-handler-files`
6. `architecture/repo-ports` + `architecture/postgres-sqlc` (+ `architecture/table-partitioning` if data grows over time)
7. Prompt `write-service-code`
8. On completion: `architecture/gen-stub` → run `make gen-stub`
