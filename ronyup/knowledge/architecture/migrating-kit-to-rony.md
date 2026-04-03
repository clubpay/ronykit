Migrate a direct `kit`-based project to `rony` incrementally, not as a single rewrite.

Start with an inventory:

- identify all current `kit` touch points (service registration, contracts, route selectors, context usage, codecs, gateway/cluster wiring),
- classify each endpoint as read-only, state-changing, or integration-heavy,
- and define migration order by risk (low-risk first).

Keep behavior stable while changing structure:

- preserve API routes, method selectors, and payload schema,
- preserve error codes/messages and logging semantics,
- preserve startup/shutdown lifecycle behavior.

Use a strangler approach per service/module:

1. Introduce a `rony` service module skeleton (`service.go`, `module.go`, `api/service.go`, `internal/app`, `internal/repo`).
2. Keep existing business logic in place first; adapt only transport and wiring layers.
3. Move handler logic from old `kit` contracts into thin `api` methods that delegate to `internal/app`.
4. Move persistence calls behind `internal/repo/port.go` interfaces, then implement adapters in `internal/repo/v0`.
5. Replace ad-hoc configuration loading with typed `internal/settings` and `x/settings.Unmarshal`.
6. Replace direct logger/tracer wiring with `x/telemetry/logkit` and `x/telemetry/tracekit`.
7. Regenerate and adopt service stubs (`make gen-stub`) for inter-service calls.
8. Remove old `kit` wiring only after parity tests pass.

Migration plan (recommended phases):

Phase 0 - Baseline and safety rails

- capture current behavior with integration tests and representative fixtures,
- add request/response golden cases for critical contracts,
- define rollback toggle (for example, keep old service entrypoint behind config/env).

Phase 1 - Bootstrap `rony` module

- scaffold the target feature using `ronyup`,
- wire `module.go` with `x/di` and datasource initializers,
- keep old implementation callable so both paths can run side-by-side in non-prod.

Phase 2 - Contract and handler migration

- recreate contracts in `api/service.go` and `api/api_*.go`,
- keep handlers thin; delegate to app layer and wrap errors with `rony/errs`,
- ensure route selectors and input/output payload compatibility.

Phase 3 - Domain and repository migration

- define domain types in `internal/domain` (no transport tags),
- define repository interfaces in `internal/repo/port.go`,
- move data access into `internal/repo/v0` (prefer Postgres + sqlc by default).

Phase 4 - Cross-cutting concerns

- move settings to typed structs (`internal/settings`),
- migrate observability to `x/telemetry` packages,
- add `x/apidoc` generation and optional `x/i18n` / `x/cache` if needed.

Phase 5 - Cutover and cleanup

- run old and new paths against the same test corpus,
- shadow traffic in staging when possible,
- switch production traffic to the `rony` path,
- remove legacy `kit`-only modules, dead contracts, and obsolete wiring.

Definition of done for each migrated service:

- all previous contracts are available through `rony`,
- integration tests pass with no behavior regressions,
- logs/traces/metrics are emitted through `x/telemetry`,
- generated stubs and API docs are up-to-date,
- legacy `kit` routing for that service is removed.
