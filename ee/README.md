# ee

A typed, event-driven entity runtime. Requires Go 1.27 (generic methods).

An **entity** is one versioned persisted record (not necessarily a rich domain
object). An **aggregate** is a loading and save boundary over several entities.

```go
type Order struct {
	ID       string `ee:"id"`
	Merchant string
	Status   string
}

func (Order) EntityName() string { return "order" }

type OrderPaid struct{ OrderID, Merchant string }

func (OrderPaid) EventName() string { return "order.paid" }
func (e OrderPaid) Key() string     { return e.OrderID }                     // optional
func (e OrderPaid) Labels() ee.L    { return ee.L{"merchant": e.Merchant} } // optional

env := ee.MustNew(ee.WithBus(b), ee.WithStore(s))

orders, _ := ee.NewEntityController[Order](env)
_ = orders.Action(func(ctx *ee.ActionCtx[Order, OrderPaid]) error {
	o, err := ctx.Entity() // loads the order whose ID == event Key()
	if err != nil {
		return err
	}

	o.Status = "paid"
	if err := ctx.Save(o); err != nil { // explicit, unless the env uses ee.WithAutoSave()
		return err
	}

	return ctx.Emit(ReceiptRequested{OrderID: o.ID}) // outbox, same transaction
}, ee.Filter(ee.Match("merchant", "m1")))

for o, err := range orders.Find(ctx, ee.Where("Merchant", "m1")) { ... }

// A read that must see a write just committed, when the store has replicas:
paid, err := orders.Get(ee.WithPrimary(ctx), id)
```

Rules enforced at registration:

- every event has exactly one consuming action per Env;
- entity names and controller names are unique;
- `Filter` requires the event to implement `Labeled`;
- aggregate actions require a keyed event (checked at compile time);
- aggregate fields must be registered entities or tagged `ee:"-"`.

## Layout

| Path                    | Contents                                                             |
|-------------------------|----------------------------------------------------------------------|
| `ee/`                   | Typed API: `Env`, controllers, actions, aggregates, queries          |
| `ee/bus`                | EventBus contract (`Bus`, `Envelope`, optional `Requester`)          |
| `ee/bus/bustest`        | Conformance suite every bus backend runs                             |
| `ee/store`              | Storage contract (`Store`, `Tx`, `View`, `Record`, `Query`, `Outbox`) |
| `ee/store/storetest`    | Conformance suite every store backend runs                           |
| `ee/mem`                | In-memory bus and store, for tests and small single-process setups   |
| `std/ee/bus/nats`       | NATS JetStream bus, with request/reply (own `go.mod`)                |
| `std/ee/bus/kafka`      | Kafka bus on franz-go, no request/reply (own `go.mod`)               |
| `std/ee/store/postgres` | Postgres store, primary/replica, sqlc (own `go.mod`)                 |
| `std/ee/store/redis`    | Redis store, primary/replica, Lua commits (own `go.mod`)             |
| `std/ee/metrics`        | OpenTelemetry counters and outbox gauges for `Env` (own `go.mod`)    |

Backends expose their own bootstrap, migration and query-optimization hooks
(e.g. Postgres migrations, Redis per-kind write/find overrides); those are not
part of the `store` contract.

Files in `ee/`:

| File                  | Responsibility                                                   |
|-----------------------|------------------------------------------------------------------|
| `env.go`              | `Env`, options, registry, `Check`/`Start`/`Close`, `Stats`, `Send` |
| `schema.go`           | `Entity`, entity schema reflection (`ee:"id"`, queryable fields) |
| `event.go`            | `Event`, `Keyed`, `Labeled`, `Meta`, send options, envelopes     |
| `selector.go`         | Label selectors for action filters                               |
| `query.go`            | `Where(...).And(...)` builder, validated and converted per field |
| `repo.go`             | Unit of work and `Repo[S]` (Get/Find/Create/Save/Delete)         |
| `action.go`           | Action registration, `Filter`, `ActionCtx[S, E]`                 |
| `entity_controller.go` | `EntityController[S]`                                             |
| `aggregate.go`        | `Aggregate`, `AggregateController[A]`, `LoadCtx`, `AggregateCtx` |
| `runtime.go`          | Unit of work + conflict retry, dispatch, outbox relay            |
| `middleware.go`       | `Middleware`, `Delivery`, `WithMiddleware`, `Dedup`, `ExpireDedup` |
| `reply.go`            | `Request`, action `Reply`                                        |
| `observe.go`          | `Observation`, `Stats`, `WithObserver`                           |
| `codec.go`            | `Codec` (JSON default)                                           |

## Runtime semantics

- **Unit of work.** Each action, and each controller `Create`/`Update`/`Delete`,
  runs in one store transaction. Saves and deletes are written immediately
  (so save → read → save works); emitted events go to the outbox in the same
  transaction and are published by the relay only after commit.
- **Optimistic concurrency.** Every entity has a version. `Save`/`Delete` only
  accept entities loaded in the same unit of work and fail with `ErrConflict` if
  someone else wrote first; the whole unit is then re-run (`WithMaxRetries`,
  default 5). Aggregates always write their root, so concurrent saves of one
  aggregate serialize.
- **Reads.** Direct `EntityController.Get`/`Find` and `AggregateController.Get`
  run in a read-only `View`, which a backend may serve from a replica (and so
  may lag). Actions and all writes use the primary.
- **Backend escape hatch.** Handler contexts expose `Tx()`, the store
  transaction, so backend-specific code (e.g. sqlc via `postgres.PgxTx`) can
  run in the same unit of work.
- **Delivery.** At-least-once. A handler error makes the bus redeliver; an
  undecodable payload is logged and dropped. Expired events and events not
  matching the action's `Filter` are acknowledged without running the handler.
- **Outbox relay.** Woken by every local commit and polled (`WithOutboxPoll`,
  default 1s) for entries left by other replicas. Entries are leased
  (`WithOutboxLease`, default 30s), published in batches (`WithOutboxBatch`,
  default 100), then acknowledged. Every replica relays, yet per-key order
  holds: a store's `Claim` never hands out an entry while an older one with
  the same partition key is still unacknowledged. `Env.Stats` reports the
  backlog when the store implements `store.StatsReporter` (mem and Postgres).
- **Unsaved changes.** With the default (no auto-save), forgetting `Save`
  drops the mutation at commit. `WithRequireSave` fails that unit of work
  instead. It cannot be combined with `WithAutoSave`.
- **Reads after writes.** Direct `Get`/`Find` use a read-only view, which
  Postgres may serve from a replica. `ee.WithPrimary(ctx)` forces that one
  read onto the primary. Actions always use the primary.
- **Request/reply.** `ee.Request[R](ctx, env, ev)` publishes `ev` and waits
  for the consuming action to call `ctx.Reply(v)`. The bus must implement
  `bus.Requester` (mem and NATS do; Kafka does not, and returns
  `ErrNotSupported`). The reply is not written to the store or the outbox.
- **Queries.** `Where` / `And` / `OrderBy` / `Limit` accept `Eq`, `Neq`, `Gt`,
  `Gte`, `Lt`, `Lte` and `In` on top-level fields. Postgres uses JSONB
  containment for equality (index it with `GINIndex`) and casts the JSON
  value for the other operators (string, number, bool, time). Nested fields
  are not queryable; a hot kind can install `postgres.WithFind`.
- **Signals.** `WithObserver` receives conflict retries, panics, permanent
  drops, handler errors and outbox publish/claim/ack failures. A panic rolls
  the unit back and is returned so the bus redelivers until its own
  `MaxAttempts`, then `OnDeadLetter`. An undecodable payload is logged,
  observed as `dropped`, and acknowledged. The observer must not call back
  into the same `Env`.
- **Middleware.** `WithMiddleware` wraps action handling, inside the unit of
  work and after expiry/Filter checks. `Delivery.Tx` lets a middleware write
  atomically with the handler; it may also ignore it and use its own resources.
  `ee.Dedup` skips already-handled envelope IDs with a marker written in the
  same transaction, giving exactly-once effects over at-least-once delivery.
  Markers are JSON objects `{"at":"<RFC3339>"}`. `Env.ExpireDedup(ctx, before)`
  deletes those strictly older than `before`, and markers that are not a
  timestamp. Run it on a timer, with `before` past the bus redelivery window
  (a day is a conservative start). Entity names starting with `ee.` are
  reserved for such internal kinds.

## Production

- Call `env.Check` during process start, before `Start`, so a missing bus or
  store fails in one place.
- `WithMiddleware(Dedup)` on every env whose handlers must not apply twice.
- Schedule `ExpireDedup`. `std/ee/metrics` maps `WithObserver` and
  `Env.Stats` onto an OpenTelemetry meter (`x/telemetry/meterkit`). Alert on
  outbox age and on signal kinds `dropped`, `panic`, `outbox_error` and
  repeated `conflict`.
- Set the bus `OnDeadLetter` (mem, NATS, Kafka). Panics and handler errors
  land there after `MaxAttempts`; undecodable events do not, they are dropped
  at ee after one observation.
- `WithOutboxLease` must be longer than a slow `Publish`. `WithMaxRetries`
  bounds hot-key conflicts (the returned error wraps `ErrConflict` and says
  how many attempts ran).
- Postgres: `ee.WithPrimary` for read-your-writes. Add `GINIndex` or
  `FieldIndex` for fields you filter on; use `WithFind` when JSONB is not
  enough.

## Status

Implemented: the runtime above, the in-memory backends, and under `std/ee` the
Postgres and Redis stores, the NATS and Kafka buses, and the OpenTelemetry
metrics adapter. `Env.Stats` / `OutboxStats` and `ExpireDedup` cover relay lag
and dedup retention. Request/reply is available through `ee.Request` on buses
that implement `bus.Requester`.

Kafka has no request/reply. Redis `Find` scans a kind's id set, so large kinds
belong in Postgres (or behind a purpose-built index).
