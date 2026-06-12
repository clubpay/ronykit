Integration tests for the repo layer are MANDATORY, not optional.

- Every method on every repository port MUST have an integration test exercising it against a real datastore (Gnomock-backed Postgres/Redis
  via `x/testkit`), not a mock or fake.
- Cover the meaningful paths per method: happy path, not-found / empty result, and constraint/conflict errors (unique violations, etc.).
- After writing them, RUN them and confirm they pass before considering the repo done — `go test ./internal/repo/...` (or `make test`). Do not
  report a repo method as implemented until its integration test has actually been executed and is green.

Write integration tests in `internal/repo/integration_test/` using `x/testkit`.

Create a `setup_test.go` with:

- `Setup(t *testing.T, populates ...any)` helper that calls `testkit.Run(...)` with:
  - `fx.Supply(set)` — a manually constructed `*settings.Settings`,
  - `fx.Provide(cache.New)`,
  - `testkit.InitDB("db", testkit.InitDBParams{User, Pass, DB, Queries: testkit.FolderContent("../v0/data/db/migrations")})`,
  - `testkit.InitRedis("redis", testkit.InitRedisParams{})`,
  - `v0repo.Init`,
  - `fx.Populate(populates...)`.

Each test file calls `Setup(t, &accountRepo)` to get the concrete repo interface injected.

Use Gnomock containers (Postgres, Redis) spun up by testkit under the hood. Keep test files in the `integration_test` package to test through the repo interface boundary.
