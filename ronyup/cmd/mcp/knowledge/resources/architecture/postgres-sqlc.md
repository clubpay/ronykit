Default persistence to Postgres with sqlc. Write the repo layer's SQL as sqlc queries — never hand-write `database/sql` query strings or use
an ORM for the v0 adapter. The DAO/`Queries` code is generated from those `.sql` files; do not hand-edit generated files.

- Place `sqlc.yml` at `internal/repo/v0/sqlc.yml` pointing schema to `data/db/migrations` and queries to `data/db/queries`.
- Use `engine: postgresql`, `emit_interface: true`, and generate into `data/db`.
- Number migration files sequentially: `001_init.up.sql`, `002_feature.up.sql`, etc.
- Put SQL queries in `data/db/queries/*.sql`, one file per domain concept (`account.sql`, `ledger.sql`).

For custom column types, use sqlc overrides to map columns to Go types from `data/db/types/ext.go`.

Embed migrations at the module root:

- `//go:embed internal/repo/v0/data/db/migrations`
- `var MigrationFS embed.FS` in `migration.go`.

Pass `MigrationFS` to `di.ProvideDBParams[settings.Settings](MigrationFS)` in the `diDatasource` block.

After adding or changing any query or migration `.sql` file, run `make sqlc` to regenerate the DAO code, then build/test. Generated DAO code
must always be in sync with the `.sql` sources — committing query changes without running `make sqlc` is a bug.

For tables expected to grow without bound, plan partitioning in SDD and read `knowledge://ronyup/architecture/table-partitioning` (strategy selection, monthly/quarterly time partitions, maintenance SQL, scheduled execution).
