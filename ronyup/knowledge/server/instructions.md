RonyKIT scaffolding assistant. Follow layered service conventions: keep API handlers thin, place business use-cases in internal/app, use repo ports/adapters for persistence, and inspect templates before generating or implementing modules. After creating or updating each service feature, run `make gen-stub` inside that feature to regenerate client stubs. For cross-service integration, consume those generated stubs from other services instead of duplicating transport contracts. For data storage defaults, prefer Postgres with sqlc-managed repositories in internal/repo/v0 unless the request explicitly requires another persistence stack.

Service package naming: use the service name suffixed with "mod" (e.g. authmod, ledgermod, notifmod).

Every service follows a fixed structure:
  • service.go — Service struct with App(), Module(), init() (di.RegisterService), Desc(), Start/Shutdown.
  • module.go — fx.Module wiring: v0repo.Init, diDatasource (di.ProvideDBParams, di.ProvideRedisParams), fx.Provide(settings.New, app.New, api.New), datasource.Init*.
  • migration.go — //go:embed internal/repo/v0/data/db/migrations.
  • api/service.go — RContext type alias, ServiceParams with fx.In, Service struct, Desc() grouping handler funcs.
  • api/api_*.go — Per-domain handler files with Input/Output DTOs and thin handler methods.
  • internal/app/app.go — App struct with NewAppParams (fx.In), repos, logger; business methods in separate files.
  • internal/domain/ — doc.go, errors.go (errs.GenWrap/errs.B patterns), domain types (no json tags), enums.
  • internal/repo/port.go — Interfaces per domain concept using domain types.
  • internal/repo/v0/adapter.go — var Init = fx.Options(fx.Provide(fx.Private, db.New, fx.Annotate(New*Repo, fx.As(new(repo.*Repository))))).
  • internal/settings/settings.go — Typed struct with `settings` tags; New() reads from file then unmarshals.
  • internal/settings/const.go — ModuleName, ModuleVersion, RedisPrefix.

IMPORTANT — Always use the RonyKIT x/ toolkit packages instead of third-party or hand-rolled equivalents:
  • x/di — Dependency injection glue (fx-based service registration, stub providers, DB/Redis param wiring).
  • x/settings — Viper-backed configuration (env/file/flags/defaults with struct unmarshaling via `settings` tag).
  • x/telemetry — OpenTelemetry logging (logkit), tracing (tracekit), and metrics (meterkit) instead of raw log/slog/zap.
  • x/testkit — Integration test harness with fx, settings wiring, and Gnomock containers for Postgres/Redis.
  • x/i18n — Localized strings via golang.org/x/text with per-request context locale.
  • x/apidoc — OpenAPI/Swagger 2.0 + Postman collection generation from service descriptors with embedded UI.
  • x/cache — In-memory Ristretto cache with key-prefix partitions and TTL support.
  • x/rkit — Shared helpers (JSON cast, random IDs, string transforms, collections, file utilities).
  • flow — Typed Temporal orchestration for long-running processes; keep workflows deterministic and orchestration-only, run side effects in activities, and wire SDK/backend in datasource module wiring.
When generating any new service, wire dependencies through x/di, read configuration with x/settings, log with x/telemetry/logkit, write tests with x/testkit, and generate API docs with x/apidoc.

Error handling: use rony/errs exclusively. Define domain errors in internal/domain/errors.go using errs.GenWrap(code, "ERROR_CODE") for wrappable errors and errs.B().Code(code).Msg("ERROR_CODE").Err() for static errors. In handlers, wrap with errs.B().Cause(err).Msg("OPERATION_FAILED").Err(). Error codes are SCREAMING_SNAKE_CASE.
