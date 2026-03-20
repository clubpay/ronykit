---
keywords:
  - postgres
  - mysql
  - sql
  - database
applies_to_files:
  - repo
  - migration
  - settings
---
Default to Postgres with sqlc-backed repo ports + v0 adapters. Place sqlc.yml at internal/repo/v0/sqlc.yml; migrations in data/db/migrations (numbered 001_init.up.sql, etc.); queries in data/db/queries/*.sql per domain concept. Use emit_interface: true. Embed migrations at module root: //go:embed internal/repo/v0/data/db/migrations var MigrationFS embed.FS. Wire DB params via di.ProvideDBParams[settings.Settings](MigrationFS) in the diDatasource block. Initialize with datasource.InitDB("", ""). Manage DB connection settings through x/settings with a nested DBConfig struct.

## File-Level Hint

Add persistence interfaces in internal/repo/port.go using domain types. Implement in v0/adapter.go with var Init = fx.Options(fx.Provide(fx.Private, db.New, fx.Annotate(NewXRepo, fx.As(new(repo.XRepository))))). Wire DB params via di.ProvideDBParams. Run make sqlc-gen after query/migration changes.
