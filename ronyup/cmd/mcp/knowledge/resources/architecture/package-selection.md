Package selection is a hard rule, not a preference. RonyKIT ships batteries-included toolkit packages under `x/`, plus `flow` and `rony/errs`. Using them — instead of the standard library or third-party equivalents — keeps every service consistent, observable, deterministic, and upgradeable from one place. Before you write a utility, a conversion, an ID generator, a logger, a config reader, a cache, a rate limiter, a batcher, a pool, a DB/Redis/S3 connection, a workflow, or an inter-service client, assume RonyKIT already has it and look first.

## Why this matters

- **Consistency**: every service behaves the same way; reviewers and tooling can rely on it.
- **Observability**: `logkit`/`tracekit`/`meterkit` are OpenTelemetry-bridged; raw `log`/`slog`/`zap` bypass the telemetry pipeline.
- **Determinism & safety**: `flow` enforces the workflow/activity split and deterministic execution that the bare Temporal SDK does not.
- **Single upgrade point**: bug fixes and perf improvements (zero-copy casts, pooled buffers, tuned DB pools) land once in `x/` and benefit every service.
- **Less surface area**: fewer third-party deps to audit, version, and secure.

## Reach-for-X → use-Y mapping

| If you reach for…                                            | Use instead                                                                                               | Resource                                                             |
|--------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------|
| `crypto/rand`, `math/rand`, `github.com/google/uuid` for IDs | `rkit.RandomID(n)`, `rkit.RandomIDs(...)`, `rkit.RandomDigit(n)`, `rkit.SecureRandomUint64()`             | `packages/rkit`                                                      |
| `encoding/json` Marshal/Unmarshal in code                    | `rkit.ToJSON` / `rkit.FromJSON` / `rkit.ToJSONStr` / `rkit.CastJSON` / `rkit.ToMap`                       | `packages/rkit`                                                      |
| manual `[]byte(s)` / `string(b)` conversions                 | `rkit.B2S` / `rkit.S2B` (zero-copy), `rkit.CloneStr` / `rkit.CloneBytes`                                  | `packages/rkit`                                                      |
| `strconv.ParseInt/FormatInt/ParseFloat`                      | `rkit.StrToInt64`/`Int64ToStr`/`StrToFloat64`/`Float64ToStr` (and the int32/uint variants)                | `packages/rkit`                                                      |
| hand-rolled case conversion                                  | `rkit.ToCamel` / `ToLowerCamel` / `ToSnake` / `ToScreamingSnake` / `ToKebab`                              | `packages/rkit`                                                      |
| `for` loops to map/filter/reduce/paginate slices             | `rkit.Map` / `rkit.Filter` / `rkit.Reduce` / `rkit.Paginate` / `rkit.ForEach`                             | `packages/rkit`                                                      |
| manual `Contains`/dedupe/slice↔map/set building              | `rkit.Contains`/`ContainsAny`/`AddUnique`/`ArrayToMap`/`ArrayToSet`/`MapToArray`                          | `packages/rkit`                                                      |
| nil-checks, deref, first-non-zero, `must` panics             | `rkit.PtrVal`/`ValPtr`/`ValPtrOrNil`/`Coalesce`/`Must`/`Ok`/`OkOr`/`Assert`                               | `packages/rkit`                                                      |
| `github.com/jinzhu/copier` directly for struct mapping       | `rkit.DynCast` / `rkit.DynCastOption` / `rkit.TypeConvert`                                                | `packages/rkit`                                                      |
| `log`, `log/slog`, `go.uber.org/zap`                         | `x/telemetry/logkit` (inject `*logkit.Logger` via fx)                                                     | `packages/logkit`, `architecture/logging`                            |
| raw OpenTelemetry tracer/meter setup                         | `x/telemetry/tracekit` / `x/telemetry/meterkit`                                                           | `packages/tracekit`, `packages/meterkit`                             |
| `os.Getenv`, `flag`, raw `spf13/viper`                       | `x/settings` (typed struct with `settings` tags)                                                          | `packages/settings`, `architecture/settings-config`                  |
| package-level globals / hand-wired constructors              | `x/di` + `uber/fx` (`di.RegisterService`, `di.ProvideDBParams`, `di.StubProvider`)                        | `packages/di`, `architecture/module-wiring`                          |
| `errors.New`, `fmt.Errorf`, third-party error libs           | `rony/errs` (`errs.GenWrap`, `errs.B().Code(...).Msg(...).Err()`)                                         | `architecture/error-handling`                                        |
| `go.temporal.io/sdk/*` for workflows                         | `flow` ONLY (never the raw Temporal SDK in service code)                                                  | `packages/flow`, `characteristics/workflow`, prompt `write-workflow` |
| `sql.Open`, `redis.NewClient`, `golang-migrate`, minio setup | `x/datasource` (`InitDB`/`InitRedis`/`InitS3`/`InitMinio*`) via `di.ProvideDBParams`/`ProvideRedisParams` | `packages/datasource`, `architecture/postgres-sqlc`                  |
| hand-rolled / third-party rate limiting                      | `x/ratelimit` (Redis-backed `Limiter`, `PerSecond`/`PerMinute`/`PerHour`)                                 | `packages/ratelimit`                                                 |
| `sync.Map`+TTL hacks, third-party in-mem caches              | `x/cache` (Ristretto, key-prefix partitions, TTL)                                                         | `packages/cache`, `characteristics/cache`                            |
| hand-rolled request coalescing / micro-batching              | `x/batch` (`NewMulti`, `Enter`/`EnterAndWait`)                                                            | `packages/batch`                                                     |
| custom `sync.Pool` for timers/waitgroups/byte buffers        | `x/p` (`AcquireTimer`/`AcquireWaitGroup`/`Bytes` pools)                                                   | `packages/p`                                                         |
| hand-rolled localization / message catalogs                  | `x/i18n` (per-request locale via context)                                                                 | `packages/i18n`, `characteristics/i18n`                              |
| hand-written OpenAPI/Swagger/Postman                         | `x/apidoc` or `rony.WithAPIDocs` on the server                                                            | `packages/apidoc`, `architecture/apidoc-generation`                  |
| hand-written HTTP/gRPC clients to other services             | generated client stubs (`make gen-stub`) consumed via `di.StubProvider`                                   | `architecture/inter-service-stubs`, `architecture/gen-stub`          |
| ad-hoc docker/test setup for integration tests               | `x/testkit` (fx + settings + Gnomock containers)                                                          | `packages/testkit`, `architecture/integration-tests`                 |

## How to comply

1. Default to the RonyKIT package; only fall back to stdlib/third-party when there is genuinely no RonyKIT equivalent — and note why.
2. If you think a needed helper is missing, search `knowledge://ronyup/packages/*` and the `x/` source before reimplementing.
3. The scaffolded workspace ships a `.golangci.yml` with `depguard` rules that fail `make lint` when forbidden imports (e.g. `go.temporal.io/sdk`, `log`, `log/slog`, `go.uber.org/zap`, `github.com/google/uuid`) appear in non-test service code. Treat lint failures here as design violations, not formatting nits.
