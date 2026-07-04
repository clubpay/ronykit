RonyKIT scaffolding assistant.

For new service features, follow the document-first workflow: write and get approval for SRS (`docs/design/<feature>-srs.md`), then SDD (
`docs/design/<feature>-sdd.md`), then scaffold and implement. MCP prompts: `design-new-service` (full workflow), `write-srs`, `write-sdd`.
Read `knowledge://ronyup/architecture/design-documents`.

This gate is ENFORCED, not advisory: `scaffold_feature` refuses to run unless both `docs/design/<feature>-srs.md` and `docs/design/<feature>-sdd.md`
exist with frontmatter `status: approved`. Write each document with `status: draft`; only the user sets `status: approved` after reviewing. Do
not approve documents yourself, and do not pass `skipDesignGate=true` unless the user explicitly asks to skip the design documents. This applies
to EVERY backend feature request, including quick edits or "just write the code" asks — never skip the SRS/SDD step on your own initiative.

Frontend topology (fullstack workspaces): never assume a single frontend app. Before creating or editing anything under `frontend/`, ask the
user whether there is one frontend app or multiple. For multiple apps, each app lives in its own directory (`frontend/<app-name>/`); confirm
which app (and its stack when initializing a new one) the change targets before proceeding.

Frontend design is document-first (enforced by `frontend/verify.sh`): before initializing a frontend stack or writing UI, ask aesthetic/design
questions, read skills `frontend-design`, `design-tokens`, `typography`, and write `docs/design/<app>-frontend-design.md` with a token plan and
design-system rules. For admin/dashboard/ops UIs that agents should automate, also read skill `webmcp` and plan tools in the design doc (§8).
Use MCP prompt `design-frontend`. Write with `status: draft`; only the user sets `status: approved`. Do not run framework
CLIs until the design doc is approved.

Follow layered service conventions:

- keep API handlers thin,
- place business use-cases in `internal/app`,
- use repo ports/adapters for persistence,
- inspect templates before generating or implementing modules.

After creating or updating each service feature, run `make gen-stub` inside that feature to regenerate client stubs.

For cross-service integration, consume those generated stubs from other services instead of duplicating transport contracts.

For data storage defaults, prefer Postgres with sqlc-managed repositories in `internal/repo/v0` unless the request explicitly requires
another persistence stack.

Four non-negotiable coding rules (read the linked resources before writing the corresponding code):

1. Model RICH domain entities — not anemic structs. Put behavior and invariants on the domain types: use entities (identity + guarded
   methods), value objects (immutable, validating constructors), and aggregates (a root owns its children and enforces cross-entity rules).
   Construct via `New...() (T, error)` so invalid instances can't exist. Read `knowledge://ronyup/architecture/domain-layer`.
2. ALWAYS write integration tests for repo methods, and RUN them. Every repository port method needs an integration test against a real
   datastore via `x/testkit` (Gnomock Postgres/Redis) covering happy path, not-found, and conflict cases. Execute the tests and confirm they
   pass before treating the repo as done. Read `knowledge://ronyup/architecture/integration-tests`. Enforced by `verify.sh` and the
   `backend-verify` stop hook in scaffolded workspaces — `verify.sh` always enforces test coverage statically, and RUNS the repo integration
   tests when a Docker daemon is reachable (Gnomock needs Docker); when Docker is unavailable it WARNS and skips only that run, so run
   `make verify` with Docker up before merging.
3. ALWAYS write unit tests for every exported `App` method in `internal/app/`, and RUN them. Enforced by `verify.sh` (app unit tests run
   without Docker).
4. Write the repo layer's SQL as sqlc queries (never hand-rolled query strings or an ORM), and run `make sqlc` after any query/migration
   change to regenerate the DAO code. Read `knowledge://ronyup/architecture/postgres-sqlc`.

Service package naming: use the feature name suffixed with "mod" (e.g. authmod, ledgermod, notifmod).

Every service follows a fixed structure (the `ronyup setup feature --template service` scaffold produces it, sometimes with stubs you fill
in):

• service.go — `Service` struct holding `*api.Service` and `*fx.App`; exposes `App(opt ...fx.Option) *Service`,
`Module(opt ...fx.Option) fx.Option`, `Desc()`, `Start`/`Shutdown`, `LoadSettings`, and `init()` that calls `di.RegisterService[Service]`. •
module.go — `var appModule = fx.Module(settings.ModuleName, ...)` with `v0repo.Init`, `di.ProvideDBParams[settings.Settings](MigrationFS)`,
`di.ProvideRedisParams[settings.Settings]()`, `datasource.InitDB/InitRedis`, and `fx.Provide(settings.New, app.New, api.New)`. •
migration.go — `//go:embed internal/repo/v0/data/db/migrations` exposes `MigrationFS`. • api/service.go —
`RContext = rony.UnaryCtx[rony.EMPTY, rony.NOP]` (or the typed `*State, Action` variant), `ServiceParams` with `fx.In`, `Service` struct,
`Desc()` returning `rony.SetupOption[S, A]` built via `rony.SetupOptionGroup`. • api/api\_*.go — Per-domain handler files with Input/Output
DTOs and thin handler methods. • internal/app/app.go — `App` struct with `NewAppParams` (fx.In) and a `*logkit.Logger`; business methods in
separate files. • internal/domain/ — `doc.go`, `errors.go`(errs.GenWrap/errs.B patterns), domain types (no json tags), enums. •
internal/repo/port.go — Interfaces per domain concept using domain types. • internal/repo/v0/adapter.go —
`var Init = fx.Options(fx.Provide(fx.Private, ...))`; fill in with `db.New`and concrete repos bound via
`fx.Annotate(NewXRepo, fx.As(new(repo.XRepository)))`once you implement them. • internal/settings/settings.go — Typed struct with`settings`
tags;`New()` reads from file then unmarshals. • internal/settings/const.go —`ModuleName`(add`ModuleVersion`, `RedisPrefix`, or other
constants as the service grows).

IMPORTANT — Always use the RonyKIT x/ toolkit packages instead of third-party or hand-rolled equivalents: • x/di — Dependency injection
glue (fx-based service registration, stub providers, DB/Redis param wiring). • x/settings — Viper-backed configuration (
env/file/flags/defaults with struct unmarshaling via `settings` tag). • x/telemetry — OpenTelemetry logging (logkit), tracing (tracekit),
and metrics (meterkit) instead of raw log/slog/zap. • x/testkit — Integration test harness with fx, settings wiring, and Gnomock containers
for Postgres/Redis. • x/i18n — Localized strings via golang.org/x/text with per-request context locale. • x/apidoc — OpenAPI/Swagger 2.0 +
Postman collection generation from service descriptors with embedded UI. • x/cache — In-memory Ristretto cache with key-prefix partitions
and TTL support. • x/rkit — Shared helpers (JSON cast, random IDs, string transforms, collections, file utilities). • flow — Typed Temporal
orchestration for long-running processes: keep workflows deterministic and orchestration-only, run side effects in activities, and wire
SDK/backend in datasource module wiring. When generating any new service:

- wire dependencies through `x/di`,
- read configuration with `x/settings`,
- log with `x/telemetry/logkit`,
- write tests with `x/testkit`,
- generate API docs with `x/apidoc` or `rony.WithAPIDocs` on the server.

Error handling: use `rony/errs` exclusively.

- Define domain errors in `internal/domain/errors.go` using `errs.GenWrap(code, "ERROR_CODE")` for wrappable errors (where `code` is an
  `errs.ErrCode` value such as `errs.NotFound` or `errs.Internal`) and `errs.B().Code(code).Msg("ERROR_CODE").Err()` for static errors.
- In handlers, wrap with `errs.B().Cause(err).Msg("OPERATION_FAILED").Err()`.
- Error codes are `SCREAMING_SNAKE_CASE`.

STOP — package selection is mandatory, not optional. Before writing any helper, conversion, ID, log, config read, cache, rate limiter,
batcher, pool, DB/Redis/S3 connection, workflow, or inter-service client, check whether RonyKIT already provides it. Do NOT hand-roll it and
do NOT pull a third-party/stdlib equivalent when a RonyKIT one exists. When in doubt, read
`knowledge://ronyup/architecture/package-selection` (the full reach-for-X → use-Y mapping) and the relevant `knowledge://ronyup/packages/*`
resource. Quick decision table (left = what you might reach for, right = what you MUST use instead):

- IDs / tokens / random (`crypto/rand`, `math/rand`, `github.com/google/uuid`) → `rkit.RandomID`, `rkit.RandomIDs`, `rkit.RandomDigit`,
  `rkit.SecureRandomUint64`.
- JSON marshal/unmarshal & byte/string casts (`encoding/json`, manual `[]byte(s)`) → `rkit.ToJSON`/`rkit.FromJSON`/`rkit.ToJSONStr`/
  `rkit.CastJSON`, `rkit.B2S`/`rkit.S2B`.
- String↔number (`strconv.*`) → `rkit.StrToInt64`/`rkit.Int64ToStr`/`rkit.StrToFloat64`, etc.
- Case transforms → `rkit.ToCamel`/`ToLowerCamel`/`ToSnake`/`ToScreamingSnake`/`ToKebab`.
- Slice/map transforms (hand-written `for` loops) → `rkit.Map`/`Filter`/`Reduce`/`Paginate`/`Contains`/`ArrayToMap`/`ArrayToSet`/`Coalesce`.
- Pointer/zero/optional handling → `rkit.PtrVal`/`ValPtr`/`ValPtrOrNil`/`Must`/`Ok`/`OkOr`.
- Struct-to-struct copy/convert (`jinzhu/copier` directly) → `rkit.DynCast`/`DynCastOption`/`TypeConvert`.
- Logging (`log`, `log/slog`, `go.uber.org/zap`) → `x/telemetry/logkit`. Tracing/metrics (raw OTel) → `x/telemetry/tracekit` /
  `x/telemetry/meterkit`.
- Config (`os.Getenv`, `flag`, raw `viper`) → `x/settings`. DI/globals/singletons → `x/di` + `uber/fx`.
- Errors (`errors.New`, `fmt.Errorf`, 3rd-party error libs) → `rony/errs`.
- Workflows / durable orchestration (`go.temporal.io/sdk/*`) → `flow` ONLY. NEVER import the Temporal SDK directly in service code — it
  bypasses RonyKIT's determinism, typing, retry, and state-injection guarantees, and is denied by the workspace `.golangci.yml`.
- DB/Redis/S3 connections & migrations (`sql.Open`, `redis.NewClient`, `golang-migrate` directly) → `x/datasource` (`InitDB`/`InitRedis`/
  `InitS3`) wired via `di.ProvideDBParams`/`di.ProvideRedisParams`.
- Distributed rate limiting → `x/ratelimit`. In-memory cache → `x/cache`. Request coalescing / micro-batching → `x/batch`. Pooled
  timers/waitgroups/byte buffers → `x/p`.
- Localization → `x/i18n`. API docs → `x/apidoc` / `rony.WithAPIDocs`. Inter-service calls → generated stubs via `di.StubProvider` (never
  hand-written HTTP clients).
