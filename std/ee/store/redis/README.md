# ee Redis store

This module implements the [`ee`](../../../../ee) store contract with Redis. Records use JSON. The ee codec must therefore produce JSON, which is the default.

```go
s, err := redis.New(redis.WithAddr("localhost:6379")) // or EE_REDIS_ADDR / EE_REDIS_URL
env := ee.MustNew(ee.WithStore(s), ee.WithBus(b))
```

## Layout

Every key starts with a prefix. The default is `{ee}`, a Redis hash tag. This prefix keeps one store in one cluster slot. Service replicas must use the same prefix.

- `{prefix}:s:<kind>:<id>` contains `v` for the version and `d` for JSON data.
- `{prefix}:k:<kind>` contains the IDs for one kind. `Find` reads this set and filters it in memory. This design suits modest kinds.
- `{prefix}:pending` is the outbox. A monotonic sequence supplies the scores. `{prefix}:o:<id>` contains the entry and its lease.

`WithReplicaAddr` or `EE_REDIS_REPLICA_ADDR` sends `View` operations to that node. `ee.WithPrimary(ctx)` reads from the primary instead. Writes, actions, and the outbox always use the primary.

## Semantics

Each commit runs one Lua script. The script checks the versions observed by the transaction. It applies writes and outbox entries only after these checks pass.

A stale version returns `store.ErrConflict` and writes nothing. The client can detect a conflict before commit. Examples are a duplicate insert or a stale update of a row already changed in the transaction. The transaction remains usable in this case. `ee.Dedup` relies on both behaviors.

`Claim` also runs one script. It leases the oldest entries. It does not return an entry while an older entry with the same partition key remains leased. Every replica can relay entries while per-key publication order is preserved.

## Tests

Set `EE_REDIS_TEST_ADDR` to enable the tests. Each test uses its own prefix.

Set `EE_REDIS_TEST_REPLICA_ADDR` to enable the replica routing test. Any other Redis instance is sufficient. The test checks only that replica reads do not see primary writes.

```sh
docker run -d --rm --name ee-redis -p 6379:6379 redis:7-alpine
docker run -d --rm --name ee-redis-replica -p 6380:6379 redis:7-alpine
EE_REDIS_TEST_ADDR=localhost:6379 \
EE_REDIS_TEST_REPLICA_ADDR=localhost:6380 \
go test ./...
```
