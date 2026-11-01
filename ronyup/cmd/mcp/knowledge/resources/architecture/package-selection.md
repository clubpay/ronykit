Package selection is mandatory. RonyKIT provides toolkit packages under `x/`, plus `flow` and `rony/errs`. Use these packages instead of standard-library or third-party equivalents.

Before you write a utility or infrastructure connection, check for a RonyKIT package. This rule covers conversions, IDs, logging, configuration, caching, rate limiting, batching, pools, DB/Redis/S3 connections, workflows, and inter-service clients.

## Why this matters

- **Consistency:** Services behave the same way. Reviewers and tools can rely on common behavior.
- **Observability:** `logkit`, `tracekit`, and `meterkit` connect to OpenTelemetry. Raw `log`, `slog`, and `zap` bypass the telemetry pipeline.
- **Determinism and safety:** `flow` enforces the workflow and activity split. It also enforces deterministic execution.
- **Single upgrade point:** Bug fixes and performance improvements in `x/` benefit every service.
- **Less surface area:** Fewer third-party dependencies require audit, versioning, and security work.

## Package mapping

| Instead of | Use | Resource |
|------------|-----|----------|
| `crypto/rand`, `math/rand`, `github.com/google/uuid` for IDs | `rkit.RandomID(n)`, `rkit.RandomIDs(...)`, `rkit.RandomDigit(n)`, `rkit.SecureRandomUint64()` | `packages/rkit` |
| `encoding/json` Marshal and Unmarshal | `rkit.ToJSON`, `rkit.FromJSON`, `rkit.ToJSONStr`, `rkit.CastJSON`, `rkit.ToMap` | `packages/rkit` |
| Manual `[]byte(s)` and `string(b)` conversions | `rkit.B2S`, `rkit.S2B`, `rkit.CloneStr`, `rkit.CloneBytes` | `packages/rkit` |
| `strconv.ParseInt`, `FormatInt`, `ParseFloat` | `rkit.StrToInt64`, `Int64ToStr`, `StrToFloat64`, `Float64ToStr`, and int32/uint variants | `packages/rkit` |
| Manual case conversion | `rkit.ToCamel`, `ToLowerCamel`, `ToSnake`, `ToScreamingSnake`, `ToKebab` | `packages/rkit` |
| `for` loops to map, filter, reduce, or paginate slices | `rkit.Map`, `Filter`, `Reduce`, `Paginate`, `ForEach` | `packages/rkit` |
| Manual `Contains`, deduplication, and slice/map/set building | `rkit.Contains`, `ContainsAny`, `AddUnique`, `ArrayToMap`, `ArrayToSet`, `MapToArray` | `packages/rkit` |
| Nil checks, dereference, first non-zero value, and `must` panics | `rkit.PtrVal`, `ValPtr`, `ValPtrOrNil`, `Coalesce`, `Must`, `Ok`, `OkOr`, `Assert` | `packages/rkit` |
| `github.com/jinzhu/copier` for struct mapping | `rkit.DynCast`, `DynCastOption`, `TypeConvert` | `packages/rkit` |
| `log`, `log/slog`, `go.uber.org/zap` | `x/telemetry/logkit` with `*logkit.Logger` injected by fx | `packages/logkit`, `architecture/logging` |
| Raw OpenTelemetry tracer or meter setup | `x/telemetry/tracekit`, `x/telemetry/meterkit` | `packages/tracekit`, `packages/meterkit` |
| `os.Getenv`, `flag`, raw `spf13/viper` | `x/settings` with typed structs and `settings` tags | `packages/settings`, `architecture/settings-config` |
| Package-level globals and manual constructors | `x/di` with `uber/fx`, `di.RegisterService`, `di.ProvideDBParams`, `di.StubProvider` | `packages/di`, `architecture/module-wiring` |
| `errors.New`, `fmt.Errorf`, third-party error libraries | `rony/errs` with `errs.GenWrap` and `errs.B()` | `packages/errs`, `architecture/error-handling` |
| `kit.Contract` and `desc.Service` in feature `api/` | `rony.SetupOptionGroup` with `WithUnary`, `WithStream`, `WithRelay` | `packages/rony`, `architecture/api-handler-files` |
| `net/http` server or an extra gateway for feature APIs | Default fasthttp gateway from `rony` | `packages/rony`, `architecture/gateway-choice` |
| Raw `kit` EdgeServer bootstrap in a scaffolded app | `rony.NewServer` in `pkg/runner` | `packages/kit`, `packages/rony` |
| `go.temporal.io/sdk/*` for workflows | `flow` only. Never use the raw Temporal SDK in service code. | `packages/flow`, `characteristics/workflow`, prompt `write-workflow` |
| `sql.Open`, `redis.NewClient`, `golang-migrate`, minio setup | `x/datasource` with `di.ProvideDBParams` and `ProvideRedisParams` | `packages/datasource`, `architecture/postgres-sqlc` |
| Manual or third-party rate limiting | `x/ratelimit` with Redis-backed `Limiter`, `PerSecond`, `PerMinute`, `PerHour` | `packages/ratelimit` |
| `sync.Map` with TTL or third-party in-memory caches | Process-local `x/cache` with Ristretto | `packages/cache`, `characteristics/cache` |
| Redis client or shared cache | `x/datasource.InitRedis`, not `x/cache` | `packages/datasource`, `characteristics/redis` |
| Manual request coalescing or micro-batching | `x/batch` with `NewMulti`, `Enter`, `EnterAndWait` | `packages/batch` |
| Custom `sync.Pool` for timers, wait groups, or byte buffers | `x/p` with `AcquireTimer`, `AcquireWaitGroup`, `Bytes` | `packages/p` |
| Manual localization or message catalogs | `x/i18n` with per-request locale in context | `packages/i18n`, `characteristics/i18n` |
| Manual OpenAPI, Swagger, or Postman generation | `x/apidoc` or `rony.WithAPIDocs` | `packages/apidoc`, `architecture/apidoc-generation` |
| Manual HTTP/gRPC clients for other services | Generated stubs from `make gen-stub`, consumed with `di.StubProvider` | `packages/stub`, `architecture/inter-service-stubs` |
| Ad-hoc Docker setup for integration tests | `x/testkit` with fx, settings, and Gnomock containers | `packages/testkit`, `architecture/integration-tests` |

## Compliance procedure

1. Use the RonyKIT package by default. Use standard-library or third-party code only when no RonyKIT equivalent exists. Record the reason.
2. Search `knowledge://ronyup/packages/*` and the `x/` source before implementing a missing helper.
3. Follow the scaffold `.golangci.yml` `depguard` rules. Forbidden imports fail `make lint` and `make verify`. Treat these failures as design violations.
4. Run `ronyup lint` or MCP tool `check_rkit_usage`. Depguard checks imports only. This linter detects manual copies of rkit helpers. `make verify` runs it on every feature. `ronyup lint --fix` rewrites supported findings.
5. Use rkit when rkit and the standard library provide the same helper. Use `rkit.Contains`, `rkit.Coalesce`, and `rkit.IntToStr` instead of `slices.Contains`, `cmp.Or`, and `strconv.Itoa`. For a plain pointer, use the `new(v)` builtin. `go fix` rewrites `rkit.ValPtr` to this builtin.
