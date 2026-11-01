RonyKIT scaffolding assistant.

Use ASD-STE100 Simplified Technical English for all responses and user-facing documentation. Use active voice, short sentences, approved words, and consistent technical terms.

## Backend design gate

For each new service feature, follow the document-first workflow:

1. Write `docs/design/<feature>-srs.md` and get user approval.
2. Write `docs/design/<feature>-sdd.md` and get user approval.
3. Scaffold and implement the feature.

Use MCP prompt `design-new-service` for the full workflow. Use `write-srs` and `write-sdd` for separate phases. Read `knowledge://ronyup/architecture/design-documents`.

The MCP `scaffold_feature` tool enforces this gate. Both design documents must exist with frontmatter `status: approved`.

Write each document with `status: draft`. Only the user sets `status: approved` after review. Do not approve documents yourself.

Do not pass `skipDesignGate=true` unless the user explicitly requests it. This rule applies to every backend feature request, including quick edits and requests to write code directly.

The direct CLI `ronyup setup feature` does not run this gate. Agents must use the MCP tool path so the gate applies.

## Design review

Use the Elephant–Goldfish Model (EGM) for design work. The primary agent drafts with project context. Fresh read-only subagents review without the drafting conversation. The user resolves findings and approves.

Read `knowledge://ronyup/architecture/egm-design-review` and `.agents/skills/ronykit-framework/references/egm.md` before requesting approval. This requirement covers SRS, SDD, frontend design, and implementation plans.

A same-thread review is not Goldfish. If fresh subagents are unavailable, mark EGM `not run`. Ask whether to continue with human-only review. EGM never replaces approval, build, test, lint, or code review gates.

## Frontend design

Never assume one frontend app in a fullstack workspace. Before creating or editing `frontend/`, ask whether there is one app or multiple apps.

For multiple apps, each app lives in `frontend/<app-name>/`. Confirm the target app before the change. Confirm its stack before initialization.

Frontend design is document-first. `check_frontend_design` and `frontend/verify.sh` enforce this rule.

Before initializing a frontend stack or writing UI:

1. Ask aesthetic and design questions.
2. Read skills `frontend-design`, `design-tokens`, `typography`, and `ui-polish`.
3. Write `docs/design/<app>-frontend-design.md` with a token plan and design-system rules.
4. For admin, dashboard, or operations UIs that agents must automate, read skill `webmcp`. Plan tools in design document section 8.
5. Use MCP prompt `design-frontend`. Use `review-frontend` to audit an existing frontend.
6. Write `status: draft`. Only the user sets `status: approved`.
7. Run `check_frontend_design` before framework CLI commands.

## Service conventions

Follow these layered service conventions:

- Keep API handlers thin.
- Put business use cases in `internal/app`.
- Use repository ports and adapters for persistence.
- Inspect templates before generating or implementing modules.

Run `make gen-stub` inside each feature after creating or updating it. Use the generated stubs for cross-service integration. Do not duplicate transport contracts.

Use Postgres with sqlc-managed repositories in `internal/repo/v0` by default. Use another persistence stack only when the request explicitly requires it.

## Mandatory coding rules

Read the linked resources before writing the corresponding code.

1. **Model rich domain entities.** Do not use anemic structs. Put behavior and invariants on domain types. Use entities with identity and guarded methods. Use immutable value objects with validating constructors. Use aggregates whose roots own children and enforce cross-entity rules. Construct with `New...() (T, error)` so invalid instances cannot exist. Read `knowledge://ronyup/architecture/domain-layer`.
2. **Write and run repository integration tests.** Every repository port method needs an `x/testkit` integration test against a real datastore. Use Gnomock Postgres or Redis. Cover the happy path, not-found, and conflict cases. Confirm that tests pass before completion. Read `knowledge://ronyup/architecture/integration-tests`. `verify.sh` and the `backend-verify` stop hook enforce this rule. Static coverage checks always run. Integration tests run when Docker is reachable. Without Docker, the gate warns and skips only that run. Run `make verify` with Docker before merging.
3. **Write and run app unit tests.** Test every exported `App` method in `internal/app/`. `verify.sh` runs these tests without Docker.
4. **Use sqlc for repository SQL.** Do not write manual query strings or use an ORM. Run `make sqlc` after query or migration changes to regenerate DAO code. Read `knowledge://ronyup/architecture/postgres-sqlc`.
5. **Keep request handling correct.** Compilation alone is insufficient. Distinguish not-found results from real failures. Take caller identity from the verified token, never the input DTO. Register authentication middleware before protected handlers. Call `StopExecution()` when middleware rejects a request. Read `knowledge://ronyup/architecture/service-correctness` before middleware, app, or repository code.

Use the feature name with suffix `mod` for the service package name. Examples are `authmod`, `ledgermod`, and `notifmod`.

## Service structure

`ronyup setup feature --template service` produces this fixed structure:

- `service.go`: Define `Service` with `*api.Service` and `*fx.App`. Expose `App`, `Module`, `Desc`, `Start`, `Shutdown`, and `LoadSettings`. Register the service in `init()` with `di.RegisterService[Service]`.
- `module.go`: Define `appModule = fx.Module(settings.ModuleName, ...)`. Include `v0repo.Init`, `di.ProvideDBParams`, `di.ProvideRedisParams`, `datasource.InitDB`, `datasource.InitRedis`, and `fx.Provide(settings.New, app.New, api.New)`.
- `migration.go`: Expose `MigrationFS` with `//go:embed internal/repo/v0/data/db/migrations`.
- `api/service.go`: Define `RContext`, `ServiceParams` with `fx.In`, and `Service`. Return `rony.SetupOption[S, A]` from `Desc()` with `rony.SetupOptionGroup`.
- `api/api_*.go`: Put each domain's handlers, input DTOs, and output DTOs in separate files. Keep handlers thin.
- `internal/app/app.go`: Define `App` with `NewAppParams` using `fx.In` and `*logkit.Logger`. Put business methods in separate files.
- `internal/domain/`: Add `doc.go`, `errors.go`, domain types without JSON tags, and enums. Use `errs.GenWrap` and `errs.B` patterns.
- `internal/repo/port.go`: Define interfaces for domain concepts with domain types.
- `internal/repo/v0/adapter.go`: Define `var Init = fx.Options(fx.Provide(fx.Private, ...))`. Include `db.New`. Bind concrete repositories with `fx.Annotate(NewXRepo, fx.As(new(repo.XRepository)))`.
- `internal/settings/settings.go`: Define a typed struct with `settings` tags. `New()` reads the file and unmarshals it.
- `internal/settings/const.go`: Define `ModuleName`. Add `ModuleVersion`, `RedisPrefix`, and other constants as necessary.

## Package selection

Always use RonyKIT `x/` packages instead of third-party or manual equivalents:

- `x/di`: Dependency injection with fx service registration, stub providers, and DB/Redis parameter wiring.
- `x/settings`: Viper-backed configuration from environment, files, flags, and defaults. Use `settings` tags for struct unmarshalling.
- `x/telemetry`: OpenTelemetry logging with logkit, tracing with tracekit, and metrics with meterkit. Do not use raw log, slog, or zap.
- `x/testkit`: Integration tests with fx, settings wiring, and Gnomock containers for Postgres and Redis.
- `x/i18n`: Localized strings with `golang.org/x/text` and per-request context locale.
- `x/apidoc`: OpenAPI/Swagger 2.0 and Postman collection generation from service descriptors with an embedded UI.
- `x/cache`: In-memory Ristretto cache with key-prefix partitions and TTL.
- `x/rkit`: Shared helpers for JSON casts, random IDs, string transforms, collections, and files.
- `flow`: Typed Temporal orchestration for long-running processes. Keep workflows deterministic and limited to orchestration. Run side effects in activities. Wire the SDK and backend in datasource module wiring.

For each new service:

- Wire dependencies with `x/di`.
- Read configuration with `x/settings`.
- Log with `x/telemetry/logkit`.
- Write tests with `x/testkit`.
- Generate API documentation with `x/apidoc` or `rony.WithAPIDocs`.

Before writing any helper or infrastructure connection, read `knowledge://ronyup/architecture/package-selection`. Read the matching `knowledge://ronyup/packages/<name>` resource. At minimum, read `rony`, `errs`, and `rkit`.

Do not import a standard-library or third-party equivalent when a RonyKIT package exists. Depguard failures from `make verify` or `golangci-lint` are design violations.

`ronyup lint` detects manual copies of rkit helpers. The MCP tool is `check_rkit_usage`. Detected patterns include error panics, default-value conditions, collection loops, dropped errors, unsafe casts, and covered standard-library calls. `make verify` runs this check on every feature. Call the tool with `fix: true` to rewrite supported findings. Use rkit when rkit and the standard library overlap.

Follow these additional rules:

- Feature `Desc()` returns `rony.SetupOptionGroup`. Never return `*desc.Service`. Never call `rony.Setup` from `api/`.
- Use `x/cache` for in-memory cache. Use `x/datasource.InitRedis` for Redis, not `x/cache`.
- Use `rony.WithRelay` and `RelayALL` for session pass-through. See `architecture/handler-relay`. Use `rony.WithReverseProxy` for a static proxy.
- Use `rony.WithUnary` for typed JSON APIs. See `architecture/gateway-choice` for gateway selection.

## Error handling

Use `rony/errs` exclusively.

- Define domain errors in `internal/domain/errors.go`. Use `errs.GenWrap(code, "ERROR_CODE")` for wrappable errors. Use `errs.B().Code(code).Msg("ERROR_CODE").Err()` for static errors.
- Wrap handler errors with `errs.B().Cause(err).Msg("OPERATION_FAILED").Err()`.
- Map only `sql.ErrNoRows` to the domain not-found error. Give every other failure its own code. Never report an arbitrary failure as `errs.NotFound`.
- Write error codes in `SCREAMING_SNAKE_CASE`.

## Workspace maintenance

For workspaces created before executable bundles, run `ronyup setup migrate bundles` once after upgrading `ronyup`. Read `knowledge://ronyup/tools/migrate_bundles`. Run it from the Go workspace root or fullstack repository root.

Use `ronyup setup sync` for `AGENTS.md`, devops, Makefile, and skill drift. By default, sync adds missing files only. Use `--overwrite` to replace existing scaffold files. Sync does not rewrite `cmd/all-in-one/main.go`.

For skill-only maintenance, use `validate_skills`. Compare installed files, frontmatter, agent links, and retirement status with the embedded catalog. Use `sync_skills` to repair drift and return validation results.

Both tools default to installed skills. Sync adds missing files only unless `overwrite: true` is explicitly requested.
