# Building Blocks: Entities, Value Objects, and Aggregates

The tactical building blocks of Domain-Driven Design provide a vocabulary for structuring domain models. Entities, Value Objects, and Aggregates are the three most critical patterns. Getting them right determines whether a domain model is expressive and maintainable or bloated and fragile.

## Entities

An Entity is a domain object defined by its identity rather than its attributes. An entity persists across time and state changes -- it is the "same thing" even when everything about it changes.

### The Identity Test

Ask: "If all the attributes change, is it still the same thing?"

- A **person** changes name, address, phone number, job, and appearance -- still the same person. **Entity.**
- A **bank account** changes its balance daily -- still the same account. **Entity.**
- A **$10 bill** is interchangeable with any other $10 bill. **Not an entity -- Value Object.**

### Identity Strategies

| Strategy | How It Works | When to Use |
|----------|-------------|-------------|
| Natural key | Use a real-world identifier (SSN, ISBN, VIN) | When a stable, unique external identifier exists |
| Surrogate key | Generate a synthetic ID (`rkit.RandomID`, UUID, auto-increment) | When no natural key exists or the natural key can change |
| Composite key | Combine multiple attributes | When identity is defined by a relationship (e.g., student + course = enrollment) |

**Prefer generated IDs over auto-increment** for distributed systems. In RonyKit, the constructor assigns the ID with `rkit.RandomID(n)` (see the scaffold's `NewItem`), so an entity has its identity before it is ever persisted; auto-increment requires a central authority and a round trip to the database.

### Entity Design Rules

1. **Identity is immutable.** Once assigned, an entity's identity never changes. If the "identity" can change, it is not really the identity.
2. **Entities are mutable.** Unlike Value Objects, entities change state over time. An `Order` moves from `Pending` to `Confirmed` to `Shipped`.
3. **Equality is based on identity.** Two `Order` values with the same `ID()` are the same order, regardless of other attribute differences. In Go, compare `a.ID() == b.ID()`, never the whole struct.
4. **Entities have a lifecycle.** They are created, go through state transitions, and may eventually be archived or deleted.
5. **Put behavior on entities.** An entity is not a data container. `order.AddLine(productID, qty, unitPrice)` belongs on the `Order` type in `internal/domain`, not in an `OrderService` or in an `*App` method that edits fields directly.

### Common Entity Pitfalls

- **Over-identification.** Making everything an entity when most things should be Value Objects. Ask the identity test for every type.
- **Anemic entities.** Structs with exported fields (or getters and setters) and no behavior. If all behavior is in `internal/app` or the handlers, the entity is a data bag.
- **Identity leakage.** Exposing database primary keys as domain identity. Use domain-meaningful identifiers (`OrderNumber()`) rather than technical ones (a `BIGSERIAL` value of `42`).

## Value Objects

A Value Object is a domain object defined entirely by its attributes. It has no identity -- two Value Objects with the same attributes are interchangeable. Value Objects are immutable: you do not change a Value Object; you replace it.

### The Attribute Test

Ask: "Is it defined by what it is, not which one it is?"

- A **mailing address** (123 Main St, Springfield, IL 62704) -- defined by its attributes. Two objects with the same street, city, state, zip are the same address. **Value Object.**
- A **money amount** ($49.99 USD) -- defined by amount and currency. **Value Object.**
- A **date range** (Jan 1 - Dec 31) -- defined by start and end. **Value Object.**
- A **color** (#FF5733) -- defined by its hex value. **Value Object.**

### Why Value Objects Matter

Value Objects are the unsung heroes of domain models. Most developers default to entities for everything, but **the majority of concepts in a well-designed domain model should be Value Objects.**

**Benefits of Value Objects:**

| Benefit | Explanation |
|---------|-------------|
| Immutability | No shared mutable state; safe to pass around, cache, and use in concurrent code |
| Side-effect-free behavior | Methods return new Value Objects rather than mutating state; easy to reason about |
| Self-validation | A Value Object validates itself on creation; an invalid Value Object can never exist |
| Equality by value | Two `Money` values with the same amount and currency are `==` in Go |
| Expressiveness | `Money` instead of `int64`; `EmailAddress` instead of `string`; domain meaning is encoded in the type |

### Value Object Design Rules

1. **Immutable.** Unexported fields set only by the constructor, exposed through getters. No setters.
2. **Self-validating.** `NewMoney` rejects a negative amount or an unknown currency by returning a domain error. If a `Money` exists, it is valid.
3. **Equality by attributes.** Go gives you this for free: a struct whose fields are all comparable (no slices, maps, or pointers) compares with `==` and can be a map key. Keep value objects comparable.
4. **Side-effect-free methods.** Value receivers that return a new value: `m.Add(other)` returns a new `Money`; it does not mutate `m`.
5. **Replace, don't modify.** To change an address, create a new `Address` and assign it: `customer.ChangeAddress(newAddress)`.

```go
type Money struct {
	amount   int64 // minor units (cents, fils); never float64
	currency Currency
}

func NewMoney(amount int64, currency Currency) (Money, error) {
	if amount < 0 {
		return Money{}, ErrMoneyNegative
	}
	if currency == CurrencyUnknown {
		return Money{}, ErrCurrencyUnknown
	}

	return Money{amount: amount, currency: currency}, nil
}

func (m Money) Amount() int64      { return m.amount }
func (m Money) Currency() Currency { return m.currency }

func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}

	return Money{amount: m.amount + other.amount, currency: m.currency}, nil
}
```

`Currency` is a type-safe enum (a struct with an unexported `slug`), so it is comparable too. The errors live in `internal/domain/errors.go`, built with `rony/errs`.

### Common Value Objects

| Value Object | Replaces | Why It Is Better |
|-------------|---------|------------------|
| `Money{amount, currency}` | `int64` / `float64` | Prevents currency mismatch errors; encapsulates rounding rules |
| `EmailAddress` | `string` | Validates format on construction; impossible to have invalid email in the system |
| `DateRange{start, end}` | Two `time.Time` fields | Enforces `start <= end`; contains overlap/contains logic |
| `Address{street, city, state, zip}` | Multiple `string` fields | Groups related data; validates as a unit |
| `Quantity{value, unit}` | `int` or `float64` | Prevents unit mismatch (adding kilograms to liters) |
| `PhoneNumber{countryCode, number}` | `string` | Validates format; normalizes representation |

### When to Use Value Objects vs. Entities

| Signal | Entity | Value Object |
|--------|--------|-------------|
| Needs to be tracked over time | Yes | No |
| Has a lifecycle (created, modified, archived) | Yes | No -- replaced, not modified |
| Two instances with same attributes are different things | Yes | No -- they are the same thing |
| Immutability is natural | No | Yes |
| Appears in the model as a measurement, description, or attribute | No | Yes |

**Rule of thumb:** If in doubt, make it a Value Object. You can always promote it to an Entity later if identity becomes important. Going the other direction (demoting an Entity to a Value Object) is much harder.

## Aggregates

An Aggregate is a cluster of domain objects (entities and value objects) treated as a single unit for data changes. Every aggregate has a single root entity -- the Aggregate Root -- through which all external access occurs.

### Why Aggregates Exist

Without aggregates, any object in the system can hold a reference to any other object and modify it directly. This creates an impossibly tangled web of dependencies where enforcing business invariants (rules that must always be true) becomes a nightmare.

Aggregates solve this by drawing a boundary:
- **Inside the boundary:** Strong consistency. All invariants are enforced within a single transaction.
- **Outside the boundary:** Eventual consistency. Changes propagate via domain events or polling.

### Aggregate Design Rules

Eric Evans and Vaughn Vernon established these rules, refined by the DDD community:

#### Rule 1: Protect Business Invariants Inside the Aggregate

An invariant is a rule that must always be true. Example: "An order's total must equal the sum of its line items." This invariant involves `Order` and `OrderLineItem`. Both belong in the same aggregate because the invariant spans both.

| If the invariant spans... | Then... |
|--------------------------|---------|
| A single entity | That entity is its own aggregate |
| An entity and its closely related objects | They form one aggregate |
| Two independently identifiable things | They are separate aggregates; enforce the rule via eventual consistency or a domain event |

#### Rule 2: Small Aggregates

Large aggregates cause:
- **Concurrency conflicts.** Two users editing different parts of the same large aggregate will conflict.
- **Performance problems.** Loading a large aggregate means loading everything it contains.
- **Transaction scope bloat.** Larger transaction scope means longer locks and more contention.

**Ideal aggregate size:** One root entity, a small set of value objects, and occasionally a small collection of child entities (e.g., `Order` with `OrderLineItems`).

**Anti-pattern:** An `Organization` aggregate that contains `Departments` which contain `Employees` which contain `Assignments`. This is too large. `Employee` should be its own aggregate, referencing `Organization` and `Department` by ID.

#### Rule 3: Reference Other Aggregates by ID Only

Do not hold direct object references to other aggregates. Instead, store only the identifier:

**Wrong:** `Order` has a field `customer *Customer` -- a direct reference to another aggregate.

**Right:** `Order` has a field `customerID string` -- a reference by ID only.

**Why:** Direct references create tight coupling, prevent independent scaling, and make it impossible to enforce aggregate boundaries. With ID references, each aggregate can be loaded, stored, and cached independently.

#### Rule 4: Use Eventual Consistency Across Aggregate Boundaries

When one aggregate's action should trigger a change in another aggregate, do not try to update both in the same transaction. Instead:

1. The first aggregate performs its action and returns (or records) a domain event
2. The app layer reacts to the event and modifies the second aggregate in a separate transaction

**Example:**
- `order.Place(now)` returns an `OrderPlaced` value; `App.PlaceOrder` saves the order
- A separate step -- another `*App` method, or a `flow` workflow when it must survive crashes and retries -- loads the `Inventory` aggregate and calls `inv.Reserve(lines)`
- These are two separate transactions (see domain-events.md)

### Choosing Aggregate Boundaries

#### Start with the Invariant

Identify every business invariant. Group objects that participate in the same invariant into the same aggregate.

**Example invariants:**
| Invariant | Objects Involved | Aggregate |
|-----------|-----------------|-----------|
| "An order total must equal the sum of its lines" | Order, OrderLineItem | Order aggregate |
| "A product must have at least one category" | Product, Category | Product aggregate (Category is a value object or ID reference) |
| "An account balance must never go below the overdraft limit" | Account | Account aggregate (single entity) |
| "A reservation cannot overlap with another for the same room" | Reservation | Reservation aggregate (overlap check is a domain service or repository query, not a cross-aggregate invariant) |

#### The Transaction Boundary Test

Ask: "Must these two changes happen atomically, or can they happen with a small delay?"

- If **atomically**: same aggregate
- If **small delay is acceptable**: separate aggregates with eventual consistency

Most of the time, a small delay is acceptable. Humans rarely need true atomicity outside of financial transactions.

### Aggregate Root Pattern

The Aggregate Root is the single entity through which all external interaction with the aggregate occurs:

**Rules for the root:**
1. External objects may only hold references to the root, never to internal entities
2. All changes to the aggregate go through the root's methods
3. The root enforces all aggregate invariants
4. The root controls the lifecycle of all internal objects
5. Delete the root and everything inside the aggregate is deleted

**Example:** in Go, the package boundary does the enforcing. `OrderLine` has unexported fields and no exported mutators, so code outside `internal/domain` can only change a line through the root:

```go
type Order struct {
	id         string
	customerID string
	status     OrderStatus
	lines      OrderLines
}

func (o *Order) AddLine(productID string, qty int, unitPrice Money) error {
	if o.status != OrderStatusPending {
		return ErrOrderNotEditable
	}
	if qty <= 0 {
		return ErrOrderLineInvalid
	}

	o.lines = append(o.lines, OrderLine{
		id:        rkit.RandomID(16),
		productID: productID,
		qty:       qty,
		unitPrice: unitPrice,
	})

	return nil
}

func (o *Order) ChangeLineQuantity(lineID string, qty int) error {
	if o.status != OrderStatusPending {
		return ErrOrderNotEditable
	}

	i := slices.IndexFunc(o.lines, func(l OrderLine) bool { return l.id == lineID })
	if i < 0 {
		return ErrOrderLineNotFound
	}
	if qty <= 0 {
		return ErrOrderLineInvalid
	}
	o.lines[i].qty = qty

	return nil
}

// Lines returns a copy so callers cannot mutate the aggregate's internals.
func (o *Order) Lines() OrderLines { return slices.Clone(o.lines) }
```

Mutating methods on the root use pointer receivers; value objects keep value receivers. Callers never write `line.qty = 5` -- they call `order.ChangeLineQuantity(lineID, 5)`.

### Common Aggregate Mistakes

| Mistake | Consequence | Fix |
|---------|------------|-----|
| Making the entire object graph one aggregate | Concurrency nightmares, slow loading | Split into multiple aggregates; reference by ID |
| Holding direct references to other aggregates | Tight coupling; cannot enforce boundaries | Replace with ID references |
| Updating multiple aggregates in one transaction | Distributed lock contention; scaling bottleneck | Use domain events (handled in a later step or a `flow` workflow) for cross-aggregate consistency |
| Putting all logic in `*App` methods instead of the aggregate root | Anemic aggregate; invariants not enforced | Move invariant-enforcing logic into the aggregate root in `internal/domain`; keep `internal/app` for orchestration |
| Creating aggregates based on database tables (or sqlc row structs) | Data model drives domain model (backward) | Design aggregates from domain invariants, then map to persistence in `internal/repo/v0` |

## Putting It All Together

A well-designed domain model has this structure:

1. **Value Objects** form the majority of types -- measurements, descriptions, identifiers, small composites
2. **Entities** represent things with identity and lifecycle -- fewer than you think
3. **Aggregates** cluster related entities and value objects behind a root -- enforcing consistency boundaries
4. **References between aggregates** are by ID only -- enabling independent evolution
5. **Cross-aggregate consistency** is achieved through domain events -- eventual consistency is the default

The result is a model that is expressive (reads like the business), consistent (invariants are enforced), and scalable (aggregates are independent units of consistency, persistence, and caching).
