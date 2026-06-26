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

Default to Postgres with sqlc-backed repo ports + v0 adapters.

- Place `sqlc.yml` at `internal/repo/v0/sqlc.yml`.
- Keep migrations in `data/db/migrations` (numbered `001_init.up.sql`, etc.).
- Keep queries in `data/db/queries/*.sql` per domain concept.
- Use `emit_interface: true`.
- Embed migrations at module root:
  - `//go:embed internal/repo/v0/data/db/migrations`
  - `var MigrationFS embed.FS`
- Wire DB params via `di.ProvideDBParams[settings.Settings](MigrationFS)` in the `diDatasource` block.
- Initialize with `datasource.InitDB("", "")`.
- Manage DB connection settings through `x/settings` with a nested `DBConfig` struct.

## File-Level Hint

Add persistence interfaces in `internal/repo/port.go` using domain types.

Implement in `v0/adapter.go` with:

- `var Init = fx.Options(fx.Provide(fx.Private, db.New, fx.Annotate(NewXRepo, fx.As(new(repo.XRepository)))))`

Wire DB params via `di.ProvideDBParams`.

Write repo SQL as sqlc queries (no hand-written query strings / ORM); run `make sqlc` after any query/migration change to regenerate the DAO
code.

## Growing data / partitioning

When SRS/SDD projects continuous growth (events, audit, ledger, logs, high-volume facts), read `knowledge://ronyup/architecture/table-partitioning` before finalizing schema. Default to PostgreSQL declarative RANGE partitioning on a timestamp, choose monthly vs quarterly granularity in SDD, and ship automated ahead-of-time partition creation plus retention drops via an idempotent SQL maintenance function. Drive it with exactly one scheduler: an in-process goroutine (run on startup + interval via fx lifecycle hook — the default), `pg_cron`, or a `flow` schedule when the module already uses `flow`.
