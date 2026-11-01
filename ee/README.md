# ee

The `ee` package is a typed, event-driven entity runtime. It requires Go 1.27 for generic methods.

An **entity** is one versioned persisted record. It does not need to be a rich domain object. An **aggregate** is a load and save boundary over several entities.

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

Registration enforces these rules:

- Every event has exactly one consuming action for each `Env`.
- Entity names and controller names are unique.
- `Filter` requires the event to implement `Labeled`.
- Aggregate actions require a keyed event. The compiler checks this requirement.
- Aggregate fields must be registered entities or have the `ee:"-"` tag.

## Layout

| Path | Contents |
|------|----------|
| `ee/` | Typed API: `Env`, controllers, actions, aggregates, and queries |
| `ee/bus` | EventBus contract: `Bus`, `Envelope`, and optional `Requester` |
| `ee/bus/bustest` | Conformance suite for every bus backend |
| `ee/store` | Storage contract: `Store`, `Tx`, `View`, `Record`, `Query`, and `Outbox` |
| `ee/store/storetest` | Conformance suite for every store backend |
| `ee/mem` | In-memory bus and store for tests and small single-process systems |
| `std/ee/bus/nats` | NATS JetStream bus with request/reply in a separate Go module |
| `std/ee/bus/kafka` | Kafka bus based on franz-go without request/reply in a separate Go module |
| `std/ee/store/postgres` | Postgres store with primary/replica and sqlc in a separate Go module |
| `std/ee/store/redis` | Redis store with primary/replica and Lua commits in a separate Go module |
| `std/ee/metrics` | OpenTelemetry counters and outbox gauges for `Env` in a separate Go module |

Backends expose their own bootstrap, migration, and query optimization hooks. Examples include Postgres migrations and Redis per-kind write and find overrides. These hooks are not part of the `store` contract.

Files in `ee/`:

| File | Responsibility |
|------|----------------|
| `env.go` | `Env`, options, registry, `Check`, `Start`, `Close`, `Stats`, and `Send` |
| `schema.go` | `Entity` and entity schema reflection for `ee:"id"` and queryable fields |
| `event.go` | `Event`, `Keyed`, `Labeled`, `Meta`, send options, and envelopes |
| `selector.go` | Label selectors for action filters |
| `query.go` | `Where(...).And(...)` builder with validation and conversion for each field |
| `repo.go` | Unit of work and `Repo[S]` with Get, Find, Create, Save, and Delete |
| `action.go` | Action registration, `Filter`, and `ActionCtx[S, E]` |
| `entity_controller.go` | `EntityController[S]` |
| `aggregate.go` | `Aggregate`, `AggregateController[A]`, `LoadCtx`, and `AggregateCtx` |
| `runtime.go` | Unit of work, conflict retry, dispatch, and outbox relay |
| `middleware.go` | `Middleware`, `Delivery`, `WithMiddleware`, `Dedup`, and `ExpireDedup` |
| `reply.go` | `Request` and action `Reply` |
| `observe.go` | `Observation`, `Stats`, and `WithObserver` |
| `codec.go` | `Codec` with JSON as the default |

## Runtime semantics

- **Unit of work:** Each action and controller `Create`, `Update`, or `Delete` runs in one store transaction. Saves and deletes occur immediately. Save, read, and save sequences work in the same transaction. Emitted events enter the outbox in that transaction. The relay publishes them only after commit.
- **Optimistic concurrency:** Every entity has a version. `Save` and `Delete` accept only entities loaded in the same unit of work. They return `ErrConflict` if another writer commits first. The runtime then repeats the whole unit. `WithMaxRetries` sets the limit, with default 5. Aggregates always write their root. Concurrent saves of one aggregate therefore run in sequence.
- **Reads:** Direct `EntityController.Get`, `EntityController.Find`, and `AggregateController.Get` use a read-only `View`. A backend can serve this view from a replica, which can lag. Actions and all writes use the primary.
- **Backend-specific transactions:** Handler contexts expose `Tx()`, the store transaction. Backend-specific code can run in the same unit of work. For example, sqlc can use `postgres.PgxTx`.
- **Delivery:** Delivery occurs at least once. A handler error causes bus redelivery. The runtime logs and drops an undecodable payload. It acknowledges expired events and events that do not match the action's `Filter` without running the handler.
- **Outbox relay:** Every local commit wakes the relay. Polling finds entries from other replicas. `WithOutboxPoll` sets the interval, with default 1s. Entries are leased with `WithOutboxLease`, default 30s. Publication uses batches with `WithOutboxBatch`, default 100. Entries are acknowledged after publication. Every replica relays entries. Per-key order is preserved. A store's `Claim` does not release an entry while an older entry with the same partition key remains unacknowledged. `Env.Stats` reports the backlog when the store implements `store.StatsReporter`. The memory and Postgres stores implement this interface.
- **Unsaved changes:** By default, auto-save is off. If an action omits `Save`, commit drops the mutation. `WithRequireSave` fails that unit of work instead. Do not combine it with `WithAutoSave`.
- **Reads after writes:** Direct `Get` and `Find` use a read-only view. Postgres can serve this view from a replica. `ee.WithPrimary(ctx)` forces one read to use the primary. Actions always use the primary.
- **Request/reply:** `ee.Request[R](ctx, env, ev)` publishes `ev` and waits for the consuming action to call `ctx.Reply(v)`. The bus must implement `bus.Requester`. Memory and NATS implement it. Kafka does not and returns `ErrNotSupported`. The reply is not written to the store or outbox.
- **Queries:** `Where`, `And`, `OrderBy`, and `Limit` accept `Eq`, `Neq`, `Gt`, `Gte`, `Lt`, `Lte`, and `In` on top-level fields. Postgres uses JSONB containment for equality. Add `GINIndex` to index this operation. Postgres casts the JSON value for other operators on string, number, bool, and time fields. Nested fields are not queryable. A hot kind can install `postgres.WithFind`.
- **Signals:** `WithObserver` receives conflict retries, panics, permanent drops, handler errors, and outbox publication, claim, and acknowledgment failures. A panic rolls back the unit and returns the panic. The bus redelivers until its `MaxAttempts`, then calls `OnDeadLetter`. An undecodable payload is logged, observed as `dropped`, and acknowledged. The observer must not call back into the same `Env`.
- **Middleware:** `WithMiddleware` wraps action handling inside the unit of work. It runs after expiry and `Filter` checks. `Delivery.Tx` lets middleware write atomically with the handler. Middleware can ignore this transaction and use its own resources. `ee.Dedup` skips already handled envelope IDs. It writes a marker in the same transaction. This gives exactly-once effects over at-least-once delivery. Markers are JSON objects with the form `{"at":"<RFC3339>"}`. `Env.ExpireDedup(ctx, before)` deletes markers strictly older than `before` and markers without a timestamp. Run it on a timer. Set `before` past the bus redelivery window. One day is a conservative start. Entity names that start with `ee.` are reserved for internal kinds.

## Production

- Call `env.Check` during process start and before `Start`. A missing bus or store then fails in one place.
- Use `WithMiddleware(Dedup)` on every `Env` whose handlers must not apply effects twice.
- Schedule `ExpireDedup`. `std/ee/metrics` maps `WithObserver` and `Env.Stats` to an OpenTelemetry meter with `x/telemetry/meterkit`. Alert on outbox age and signal kinds `dropped`, `panic`, `outbox_error`, and repeated `conflict`.
- Set the bus `OnDeadLetter` for memory, NATS, and Kafka. Panics and handler errors arrive there after `MaxAttempts`. Undecodable events do not arrive there. The runtime drops them after one observation.
- Set `WithOutboxLease` longer than a slow `Publish`. `WithMaxRetries` limits hot-key conflicts. The returned error wraps `ErrConflict` and reports the number of attempts.
- For Postgres, use `ee.WithPrimary` for read-your-writes behavior. Add `GINIndex` or `FieldIndex` for filtered fields. Use `WithFind` when JSONB is insufficient.

## Status

The runtime, in-memory backends, and `std/ee` adapters are implemented. These adapters include Postgres and Redis stores, NATS and Kafka buses, and OpenTelemetry metrics.

`Env.Stats`, `OutboxStats`, and `ExpireDedup` cover relay lag and deduplication retention. `ee.Request` supports request/reply on buses that implement `bus.Requester`.

Kafka does not support request/reply. Redis `Find` scans a kind's ID set. Use Postgres or a purpose-built index for large kinds.
