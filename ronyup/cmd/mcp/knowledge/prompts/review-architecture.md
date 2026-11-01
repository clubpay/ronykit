---
name: review-architecture
description: Review the architecture of an existing RonyKIT service feature for best-practice compliance.
arguments:
- name: feature_path
  description: Path to the feature directory to review.
  required: true
---

Review the RonyKIT service feature at "{{feature_path}}" for architecture compliance.

Check these rules:

1. API handlers are thin. They validate input and delegate business logic to `internal/app`.
2. `internal/repo/port.go` interfaces abstract persistence.
3. `x/di.RegisterService` wires dependencies in `module.go`. Package-level globals and singletons are absent.
4. `x/settings` supplies configuration through a typed struct and `settings` tags. Raw `os.Getenv`, `flag`, and Viper calls are absent.
5. `x/telemetry/logkit` supplies all logging. Raw zap, slog, and log calls are absent. `x/telemetry/tracekit` and `meterkit` supply tracing and metrics.
6. Integration tests use `x/testkit` with Gnomock containers.
7. `x/apidoc` or `rony.WithAPIDocs` generates API documentation.
8. Client stubs are current after `make gen-stub`. Inter-service calls use generated stubs with `di.StubProvider`, not manual HTTP clients.
9. `rony/errs` supplies all errors through `errs.GenWrap` and `errs.B`. Codes use `SCREAMING_SNAKE_CASE`. Bare `errors.New` and `fmt.Errorf` are absent.
10. Workflows use `flow` only. Report every direct `go.temporal.io/sdk` import as a violation.
11. `x/datasource` supplies infrastructure connections through `InitDB`, `InitRedis`, and `InitS3` with di parameters. Manual `sql.Open`, `redis.NewClient`, and migration setup are absent.
12. `x/rkit` supplies common utilities. These utilities include IDs, JSON/byte casts, string/number conversions, case transforms, and collections. Manual loops, direct random or UUID packages, and raw `strconv` or `encoding/json` are absent. Run MCP tool `check_rkit_usage` and include every finding.
13. `x/ratelimit` supplies rate limiting. `x/cache` supplies in-memory caching. `x/batch` supplies batching. `x/p` supplies pooling. Third-party and manual equivalents are absent.
14. Search non-test files for forbidden imports. Report `go.temporal.io/sdk`, `log/slog`, `go.uber.org/zap`, and `github.com/google/uuid` as violations.
15. Check correctness with `knowledge://ronyup/architecture/service-correctness`:
    - Repositories map only `sql.ErrNoRows` to not-found.
    - Handlers never set `errs.NotFound` for an arbitrary error.
    - Owned resources are checked against caller identity from the token, never an input DTO field.
    - Every handler is registered after its authentication middleware in `Desc()`.
    - Middleware calls `StopExecution()` when it rejects a request.
    - Request paths do not use `context.Background()`.
    - Code uses `ON CONFLICT` instead of select-then-insert where applicable.

Use `knowledge://ronyup/architecture/package-selection` as the package reference. Report every violation and suggest a specific fix. Name the exact RonyKIT package and function to use.
