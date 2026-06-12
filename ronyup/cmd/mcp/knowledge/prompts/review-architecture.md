---
name: review-architecture
description: Review the architecture of an existing RonyKIT service feature for best-practice compliance.
arguments:
- name: feature_path
  description: Path to the feature directory to review.
  required: true
---

Review the RonyKIT service feature at "{{feature_path}}" for architecture best-practice compliance.

Check the following:

1. API handlers are thin — validation only, business logic delegated to internal/app.
2. Persistence is abstracted behind internal/repo/port.go interfaces.
3. Dependencies are wired through x/di.RegisterService in module.go (no package-level globals/singletons).
4. Configuration uses x/settings with typed struct and `settings` tags (no os.Getenv/flag/raw viper).
5. Logging uses x/telemetry/logkit exclusively (no raw zap/slog/log); tracing/metrics via x/telemetry/tracekit/meterkit.
6. Integration tests use x/testkit with Gnomock containers.
7. API docs are generated with x/apidoc (or rony.WithAPIDocs).
8. Client stubs are up to date (make gen-stub) and inter-service calls go through generated stubs (di.StubProvider), not hand-written HTTP clients.
9. Error handling uses rony/errs exclusively (errs.GenWrap / errs.B), with SCREAMING_SNAKE_CASE codes — no bare errors.New/fmt.Errorf.
10. Workflows use the `flow` package only — flag ANY direct `go.temporal.io/sdk` import as a violation.
11. Infra connections use x/datasource (InitDB/InitRedis/InitS3) via di params — no hand-written sql.Open/redis.NewClient/migrate setup.
12. Common utilities use x/rkit (IDs, JSON/byte casts, string↔number, case transforms, collection ops) instead of hand-rolled loops, crypto/rand, google/uuid, or raw strconv/encoding/json.
13. Rate limiting uses x/ratelimit; in-memory caching uses x/cache; batching uses x/batch; pooling uses x/p — not third-party or hand-rolled equivalents.
14. Forbidden imports: grep the feature for `go.temporal.io/sdk`, `log/slog`, `go.uber.org/zap`, `github.com/google/uuid` in non-test files and report each as a violation.

Use `knowledge://ronyup/architecture/package-selection` as the reference for the correct RonyKIT package per concern. Report any violations and suggest specific fixes (name the exact RonyKIT package/function to switch to).
