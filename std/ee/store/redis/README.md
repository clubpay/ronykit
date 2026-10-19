# ee Redis store

Redis implementation of the [`ee`](../../../../ee) store contract. Records are
JSON, so the ee codec must produce JSON (the default does).

```go
s, err := redis.New(redis.WithAddr("localhost:6379")) // or EE_REDIS_ADDR / EE_REDIS_URL
env := ee.MustNew(ee.WithStore(s), ee.WithBus(b))
```

## Layout

Every key starts with the prefix (default `{ee}`, a Redis hash tag) so one
store occupies one cluster slot. Service replicas must use the same prefix.

- `{prefix}:s:<kind>:<id>` holds `v` (version) and `d` (JSON).
- `{prefix}:k:<kind>` is the set of ids of that kind. `Find` reads this set
  and filters in memory, which suits modest kinds.
- `{prefix}:pending` is the outbox, scored by a monotonic sequence.
  `{prefix}:o:<id>` holds the entry and its lease.

`WithReplicaAddr` (or `EE_REDIS_REPLICA_ADDR`) sends `View` to that node.
`ee.WithPrimary(ctx)` reads the primary instead. Writes, actions and the
outbox always use the primary.

## Semantics

Commits are one Lua script: the versions the transaction observed are
checked, and only then are writes and outbox entries applied. A stale version
returns `store.ErrConflict` and writes nothing, but a conflict detected in
the client before commit (a duplicate insert, a stale update of a row already
changed in this transaction) leaves the transaction usable. `ee.Dedup` relies
on both.

`Claim` is one script. It leases the oldest entries and will not return an
entry while an older one with the same partition key is still leased, so
per-key publish order holds with every replica relaying.

## Tests

Skipped unless `EE_REDIS_TEST_ADDR` is set. Each test uses its own prefix.
`EE_REDIS_TEST_REPLICA_ADDR` enables the replica routing test; any other
Redis is enough, because the test only checks that replica reads do not see
primary writes.

```sh
docker run -d --rm --name ee-redis -p 6379:6379 redis:7-alpine
docker run -d --rm --name ee-redis-replica -p 6380:6379 redis:7-alpine
EE_REDIS_TEST_ADDR=localhost:6379 \
EE_REDIS_TEST_REPLICA_ADDR=localhost:6380 \
go test ./...
```
