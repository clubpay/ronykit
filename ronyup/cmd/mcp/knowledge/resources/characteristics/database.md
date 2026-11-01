---

keywords:
- postgres
- mysql
- sql
- database
- partition
- partitioning
- retention
- growth
- scale
- archive
applies_to_files:
- repo
- migration
- settings

---

Use Postgres with sqlc-backed repository ports and v0 adapters by default.

- Place `sqlc.yml` at `internal/repo/v0/sqlc.yml`.
- Keep migrations in `data/db/migrations`. Use numbered names such as `001_init.up.sql`.
- Keep queries in `data/db/queries/*.sql` for each domain concept.
- Use `emit_interface: true`.
- Embed migrations at module root:
  - `//go:embed internal/repo/v0/data/db/migrations`
  - `var MigrationFS embed.FS`
- Wire DB parameters with `di.ProvideDBParams[settings.Settings](MigrationFS)` in the `diDatasource` block.
- Initialize with `datasource.InitDB("", "")`.
- Manage DB connection settings with `x/settings` and a nested `DBConfig` struct.

## File-Level Hint

Add persistence interfaces in `internal/repo/port.go` with domain types.

Implement them in `v0/adapter.go` with:

- `var Init = fx.Options(fx.Provide(fx.Private, db.New, fx.Annotate(NewXRepo, fx.As(new(repo.XRepository)))))`

Wire DB parameters with `di.ProvideDBParams`.

Write repository SQL as sqlc queries. Do not use manual query strings or an ORM. Run `make sqlc` after query or migration changes to regenerate DAO code.

## Growing data and partitioning

Read `knowledge://ronyup/architecture/table-partitioning` before finalizing a schema with continuous growth. Examples include events, audit records, ledgers, logs, and high-volume facts.

Use PostgreSQL declarative RANGE partitioning on a timestamp by default. Select monthly or quarterly granularity in the SDD. Automate partition creation ahead of time and retention drops with an idempotent SQL maintenance function.

Use exactly one scheduler. The default is an in-process goroutine that runs on startup and on an interval through an fx lifecycle hook. You can also use `pg_cron`. Use a `flow` schedule only when the module already uses `flow`.
