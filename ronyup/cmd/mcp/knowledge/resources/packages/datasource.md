---
import_path: github.com/clubpay/ronykit/x/datasource
short_name: datasource
---

fx-based providers that open and health-check infrastructure connections — Postgres, Redis, S3/MinIO — and run DB migrations, from typed
params. Use this instead of calling `sql.Open`, `redis.NewClient`, `golang-migrate`, or the minio client directly: it applies tuned pool
settings, runs embedded migrations, pings on startup, and integrates with `x/di` connection params.

## Usage Hint

Wire connections in the module, never by hand:

- `datasource.InitDB(in, out)` — provides a `*sql.DB` (pgx driver). Reads `datasource.DBParams`, sets sane pool limits, and runs migrations
  from `DBParams.Migrations` (an embedded `fs.FS`).
- `datasource.InitRedis(in, out)` — provides a `*redis.Client` from `datasource.RedisParams`, with TLS hardening.
- `datasource.InitS3(in, out)` / `InitMinioClient(in, out)` / `InitMinioCore(in, out)` — provide minio clients from `S3Params`/
  `MinioParams`.

The `in`/`out` strings are fx named-tag annotations so a service can hold multiple DBs/Redises. Feed the params from settings using `x/di`:

```go
var appModule = fx.Module(settings.ModuleName,
v0repo.Init,
di.ProvideDBParams[settings.Settings](MigrationFS),
di.ProvideRedisParams[settings.Settings](),
datasource.InitDB("", ""),
datasource.InitRedis("", ""),
fx.Provide(settings.New, app.New, api.New),
)
```

`MigrationFS` is exposed via `//go:embed internal/repo/v0/data/db/migrations` in `migration.go`. Do not write your own connection bootstrap
or migration runner.
