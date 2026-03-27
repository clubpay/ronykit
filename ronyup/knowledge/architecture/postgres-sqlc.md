Default persistence to Postgres with sqlc.

- Place `sqlc.yml` at `internal/repo/v0/sqlc.yml` pointing schema to
  `data/db/migrations` and queries to `data/db/queries`.
- Use `engine: postgresql`, `emit_interface: true`, and generate into `data/db`.
- Number migration files sequentially: `001_init.up.sql`, `002_feature.up.sql`,
  etc.
- Put SQL queries in `data/db/queries/*.sql`, one file per domain concept
  (`account.sql`, `ledger.sql`).

For custom column types, use sqlc overrides to map columns to Go types from
`data/db/types/ext.go`.

Embed migrations at the module root:

- `//go:embed internal/repo/v0/data/db/migrations`
- `var MigrationFS embed.FS` in `migration.go`.

Pass `MigrationFS` to `di.ProvideDBParams[settings.Settings](MigrationFS)` in
the `diDatasource` block.

Run `make sqlc-gen` to regenerate after query/migration changes.
