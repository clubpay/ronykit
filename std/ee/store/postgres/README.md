# ee Postgres store

Postgres implementation of the [`ee`](../../../../ee) store contract, on pgx
and sqlc.

```go
pg, err := postgres.New(
	postgres.WithPrimaryDSN(primaryDSN),
	postgres.WithReplicaDSNs(replica1, replica2),
	postgres.WithMigration("001-order-indexes", func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, postgres.GINIndex("order"))
		return err
	}),
)
if err := pg.Migrate(ctx); err != nil { ... } // tables + pending migrations
if err := pg.Check(ctx); err != nil { ... }   // fail fast at boot

env := ee.MustNew(ee.WithStore(pg), ee.WithBus(b))
```

Without pool/DSN options, `EE_POSTGRES_PRIMARY_DSN` and
`EE_POSTGRES_REPLICA_DSNS` (comma separated) are used.

## Primary and replicas

| Operation                                                  | Pool                    |
|------------------------------------------------------------|-------------------------|
| actions, `Create`/`Update`/`Delete`, outbox, migrations    | primary                 |
| `StateController.Get`/`Find`, `AggregateController.Get`    | replicas, round robin   |

Replica reads can lag. For read-your-writes, force the primary for one call:
`orders.Get(postgres.ForcePrimary(ctx), id)`. With no replicas, everything
uses the primary.

## Tables

One JSONB table for every state kind (`ee_states`), the outbox (`ee_outbox`)
and applied migration names (`ee_migrations`); see [`sql/schema.sql`](sql/schema.sql).
The ee codec must produce JSON (the default does).

## Queries and indexes

`Find` is generated per query, with kind and field names inlined as escaped
literals and values bound as parameters, and streamed through a server-side
cursor (`WithFetchSize`, default 100), so a handler may write while iterating.

- Equality (`ee.Where(field, v)`) uses JSONB containment; `GINIndex(kind)`
  serves it for every field of the kind.
- Ranges and `In` on text/numeric fields: `FieldIndex(kind, field, Text|Numeric)`
  indexes exactly the expression `Find` generates (verified against the
  planner in `find_internal_test.go`). Field is the JSON name.

To replace the generated query for a kind, use `WithFind(kind, fn)`.

## sqlc

The store's own SQL lives in `sql/` and is generated into `internal/db`
(`go generate ./...`). Your own sqlc code can join an ee transaction:

```go
orders.Action(func(ctx *ee.ActionCtx[Order, OrderPaid]) error {
	ptx, _ := postgres.PgxTx(ctx.Tx())
	open, err := mydb.New(ptx).CountOpenOrders(ctx, ctx.Event().Merchant)
	...
})
```

`Store.Primary()` exposes the write pool for sqlc code outside ee.

## Semantics

- Optimistic writes are version-checked `UPDATE`/`DELETE`s and
  `INSERT ... ON CONFLICT DO NOTHING`, so a stale version or duplicate insert
  returns `store.ErrConflict` without aborting the transaction (`ee.Dedup`
  relies on this). Serialization failures and deadlocks also map to
  `ErrConflict`, so ee retries the unit of work.
- `Claim` serializes claimers with an advisory lock and skips an entry while an
  older one with the same partition key is leased: per-key publish order holds
  with every replica relaying.

## Tests

Integration tests are skipped unless `EE_POSTGRES_TEST_DSN` is set; they
truncate the ee tables. `EE_POSTGRES_TEST_REPLICA_DSN` enables the replica
routing test (any other database works). The store also runs the shared
conformance suite, `ee/store/storetest`.

```sh
docker run -d --rm --name ee-pg -e POSTGRES_PASSWORD=ee -e POSTGRES_USER=ee -p 55432:5432 postgres:17-alpine
docker exec ee-pg psql -U ee -c 'CREATE DATABASE ee_replica'
EE_POSTGRES_TEST_DSN='postgres://ee:ee@localhost:55432/ee?sslmode=disable' \
EE_POSTGRES_TEST_REPLICA_DSN='postgres://ee:ee@localhost:55432/ee_replica?sslmode=disable' \
go test ./...
```
