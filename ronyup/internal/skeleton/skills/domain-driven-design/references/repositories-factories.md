# Repositories and Factories

Repositories and Factories are infrastructure-facing patterns in Domain-Driven Design that separate domain logic from persistence and object creation concerns. The Repository provides the illusion of an in-memory collection of aggregates. The Factory encapsulates complex creation logic. Together, they keep the domain model clean and focused on business rules.


## Table of Contents
1. [The Repository Pattern](#the-repository-pattern)
2. [The Factory Pattern](#the-factory-pattern)
3. [The Specification Pattern](#the-specification-pattern)
4. [Ports and Adapters Relationship](#ports-and-adapters-relationship)

---

## The Repository Pattern

A Repository mediates between the domain and data mapping layers, acting like an in-memory collection of domain objects. Domain code uses the repository to obtain aggregates without knowing how they are stored, queried, or reconstructed.

### Why Repositories Exist

Without repositories, domain logic becomes tangled with data access:

```go
// Without repository -- the use case is polluted with SQL and row scanning.
func (a *App) ApproveClaim(ctx context.Context, claimID string) error {
	var status string
	var amount int64
	err := a.db.QueryRowContext(ctx,
		"SELECT status, amount FROM claims WHERE id = $1", claimID,
	).Scan(&status, &amount)
	if err != nil {
		return err
	}
	if status != "IN_REVIEW" {
		return domain.ErrClaimNotReviewable
	}
	_, err = a.db.ExecContext(ctx, "UPDATE claims SET status = 'APPROVED' WHERE id = $1", claimID)

	return err
}
```

```go
// With repository -- the use case reads like the business operation.
func (a *App) ApproveClaim(ctx context.Context, claimID string) error {
	claim, err := a.claims.Get(ctx, claimID)
	if err != nil {
		return err
	}
	if err := claim.Approve(a.now()); err != nil {
		return err
	}

	return a.claims.Save(ctx, claim)
}
```

The second version is readable by a domain expert. The first is not -- and it also re-implements the "only claims in review can be approved" invariant outside the `Claim` aggregate.

### Repository Interface Design

In RonyKit, the repository interface (the **port**) lives in `internal/repo/port.go`, package `repo`, and is expressed only in `internal/domain` types. `internal/app` depends on `repo`; it never imports the implementation in `internal/repo/v0`. This still satisfies the Dependency Inversion Principle: the port package depends only on `domain`, and the sqlc implementation depends on the port, so every dependency points toward the domain. Declare one interface per aggregate (or domain concept), and let it speak the ubiquitous language:

**Good repository methods:**
- `Get(ctx, orderID)` -- straightforward identity lookup
- `ListPending(ctx)` -- uses domain language ("pending")
- `ListByCustomer(ctx, customerID)` -- domain-meaningful query
- `ListOverdue(ctx, asOf time.Time)` -- business concept in the method name

**Bad repository methods:**
- `GetByStatusCode(ctx, 3)` -- magic number; what is status 3?
- `Query(ctx, sql string)` -- leaks persistence technology into the domain
- `FindAllWithJoins(ctx)` -- technical concern, not domain language
- `GetByColumn(ctx, "status", "PENDING")` -- generic data access, not domain query

Parameters and results are domain types (`domain.Invoice`, `domain.Invoices`) -- never sqlc row structs from `data/db` and never API DTOs.

### Collection-Oriented vs. Persistence-Oriented Repositories

Eric Evans described two flavors of repository, each modeling a different metaphor:

#### Collection-Oriented Repository

Models the repository as an in-memory collection. You add objects to it and remove objects from it. Changes to retrieved objects are automatically tracked and persisted (like JPA/Hibernate managed entities).

```go
type OrderRepository interface {
	Add(ctx context.Context, order domain.Order) error         // like appending to a collection
	Remove(ctx context.Context, orderID string) error          // like deleting from a collection
	Get(ctx context.Context, id string) (*domain.Order, error) // no Save: changes are auto-tracked
}
```

**Best with:** ORMs that support change tracking (JPA/Hibernate, Entity Framework).

**Advantages:** Clean domain model; changes feel natural; no explicit save calls.

**Disadvantages:** "Magic" change tracking can surprise developers; harder to reason about when persistence happens.

#### Persistence-Oriented Repository

Models the repository as a storage mechanism. You explicitly save objects and the repository does not track changes automatically.

```go
type OrderRepository interface {
	Save(ctx context.Context, order domain.Order) error // explicit insert/update
	Delete(ctx context.Context, orderID string) error
	Get(ctx context.Context, id string) (domain.Order, error)
}
```

**Best with:** Frameworks without change tracking (most non-ORM approaches, event sourcing, document stores).

**Advantages:** Explicit control over when persistence happens; no surprises; easier to test.

**Disadvantages:** Must remember to call `Save`; risk of losing changes if save is forgotten.

**Which to choose:** If your persistence technology offers change tracking and your team is comfortable with it, use collection-oriented. Otherwise, use persistence-oriented. The persistence-oriented style is more common in modern applications because it is more explicit. **In RonyKit it is the default:** the v0 adapter is sqlc over Postgres, which has no change tracking, and ORMs are not used.

### Repository Implementation

The port lives in `internal/repo`. The implementation lives in `internal/repo/v0` (package `v0repo`) and is bound to the port with fx. This is the Dependency Inversion Principle in action:

```
internal/
    domain/
        order.go              # Order aggregate root, OrderLine, OrderStatus
        errors.go             # rony/errs sentinels
    repo/
        port.go               # type OrderRepository interface { ... }  (package repo)
        v0/
            adapter.go        # fx.Annotate(NewOrderRepository, fx.As(new(repo.OrderRepository)))
            order.go          # sqlc-backed implementation (package v0repo)
            data/db/queries/order.sql
        integration_test/     # runs the adapter against real Postgres via x/testkit
    app/
        app.go                # depends on repo.OrderRepository only
        app_test.go           # uses an in-memory fake of the port
```

The port package defines what the app needs, in domain terms. The `v0` adapter provides it. Neither `domain` nor `app` imports `v0`.

### What a Repository Returns

A repository always returns fully constituted aggregates -- not partial objects, not DTOs, not database rows. The aggregate returned from a repository must be in a valid state with all its invariants satisfied.

**Correct:** `orders.Get(ctx, id)` returns a `domain.Order` with all its `OrderLine`s loaded, ready to have business operations performed on it.

**Incorrect:** `orders.Get(ctx, id)` returns a `db.Order` sqlc row (or an `OrderDTO`) with some fields populated and the lines missing. The caller must check which fields are available.

### Repository Anti-Patterns

| Anti-Pattern | Problem | Fix |
|-------------|---------|-----|
| Generic repository (`Repository[T any]`) | All aggregates look the same; domain-specific queries do not fit the generic interface | Create specific repository interfaces per aggregate type in `port.go` |
| Repository returns DTOs or sqlc rows | DTOs are not domain objects; behavior cannot be called on them | Return full aggregates; use separate read models (CQRS) for queries |
| Repository per entity (not per aggregate) | Bypasses aggregate root; allows direct modification of internal entities | One repository per aggregate root only |
| Repository with business logic | Repository starts containing validation or transformation logic | Keep repositories as pure storage (plus row/domain mapping and constraint-violation-to-domain-error mapping); domain logic belongs in the aggregate |
| Repository depends on domain services or `internal/app` | Circular dependency between app/domain services and repositories | Repositories depend only on the domain model (aggregates, value objects, domain errors) |

## The Factory Pattern

A Factory encapsulates the logic of creating a domain object, ensuring that the object is fully formed and valid from the moment it exists. In DDD, factories are used when object creation is complex enough to warrant its own abstraction.

In Go, the default factory is the constructor function: `NewX(...) (X, error)` in `internal/domain`. For most aggregates it is all you need. Reach for a separate factory function only when creation genuinely involves several inputs or a decision.

### When to Use a Factory

| Situation | Factory Needed? | Why |
|-----------|----------------|-----|
| Creating a Value Object with 2-3 fields | No | A constructor suffices: `NewMoney(10000, CurrencyUSD)` |
| Creating an aggregate with multiple parts and validation rules | Yes | The assembly logic is complex; a single constructor would be enormous |
| Creating an object from an external representation (API response, file) | Yes | Translation from external format to domain object is a separate concern (an ACL translator or the handler's DTO mapping) |
| Creating an object with conditional logic (different variants) | Yes | The decision of which variant to create should not be in client code |
| Reconstituting an object from persistence | Maybe | If the repository handles it, a separate factory may not be needed |

### Factory Patterns in DDD

#### Factory Method on the Aggregate

The most common pattern: a constructor function next to the aggregate, in the same package so it can set unexported fields:

```go
func NewOrderFromCart(cart Cart, customerID string) (Order, error) {
	if customerID == "" {
		return Order{}, ErrOrderCustomerRequired
	}
	if cart.IsEmpty() {
		return Order{}, ErrOrderEmpty
	}

	return Order{
		id:         rkit.RandomID(16),
		customerID: customerID,
		status:     OrderStatusPending,
		lines:      rkit.Map(cart.Items(), newOrderLineFromCartItem),
	}, nil
}
```

**Advantages:** Creation logic lives close to the aggregate. The aggregate controls its own birth.

#### Factory Method on Another Aggregate

When one aggregate creates another:

```go
func (q Quote) ConvertToOrder() (Order, error) {
	if q.IsExpired() {
		return Order{}, ErrQuoteExpired
	}

	lines := rkit.Map(q.lines, func(l QuoteLine) OrderLine {
		return OrderLine{id: rkit.RandomID(16), productID: l.productID, qty: l.qty, unitPrice: l.quotedPrice}
	})

	return Order{
		id:            rkit.RandomID(16),
		customerID:    q.customerID,
		status:        OrderStatusPending,
		lines:         lines,
		sourceQuoteID: q.id,
	}, nil
}
```

#### Standalone Factory

When creation logic does not naturally belong to any existing aggregate. In class-based languages this returns a different subclass per loan type; in Go the variant becomes a type-safe enum plus a small `Collateral` interface implemented by `Property` and `Vehicle`:

```go
func NewLoanApplication(sub Submission, report CreditReport) (LoanApplication, error) {
	applicant, err := NewApplicant(sub.Name(), sub.NationalID())
	if err != nil {
		return LoanApplication{}, err
	}

	var collateral Collateral
	switch sub.LoanKind() {
	case LoanKindMortgage:
		collateral = sub.Property()
	case LoanKindAuto:
		collateral = sub.Vehicle()
	default:
		return LoanApplication{}, ErrLoanKindUnsupported
	}

	return LoanApplication{
		id:         rkit.RandomID(16),
		kind:       sub.LoanKind(),
		applicant:  applicant,
		riskScore:  CalculateRiskScore(report),
		collateral: collateral,
	}, nil
}
```

### Factory Invariants

The most critical rule of factories: **a factory must never produce an invalid object.** If the inputs are insufficient or violate business rules, the factory must fail (return a domain error), not produce a partially valid object.

```go
// Good -- the constructor enforces invariants.
func NewOrder(customerID string, lines OrderLines) (Order, error) {
	if len(lines) == 0 {
		return Order{}, ErrOrderEmpty
	}
	if customerID == "" {
		return Order{}, ErrOrderCustomerRequired
	}

	return Order{id: rkit.RandomID(16), customerID: customerID, status: OrderStatusPending, lines: lines}, nil
}

// Bad -- the constructor produces potentially invalid objects:
// callers can now hold an order with no customer and no lines.
func NewOrderUnchecked(customerID string, lines OrderLines) Order {
	return Order{id: rkit.RandomID(16), customerID: customerID, lines: lines}
}
```

Exported struct fields defeat the constructor entirely (`domain.Order{}` is always possible), which is why invariant-backed fields stay unexported.

### Reconstitution vs. Creation

There is an important distinction between creating a new aggregate and reconstituting one from persistence:

| Aspect | Creation | Reconstitution |
|--------|---------|----------------|
| When | A new domain object comes into existence | An existing object is loaded from storage |
| Validation | Full business rule validation | No validation needed; data was validated on creation |
| Domain events | May raise creation events (`OrderCreated`) | Should NOT raise events; nothing new happened |
| Identity | Generate a new ID (`rkit.RandomID`) | Use the stored ID |
| Invariants | Enforce all invariants | Assume invariants hold (data was valid when stored) |

Reconstitution typically happens inside the repository implementation. Because the adapter lives in another package, `internal/domain` exports a dedicated reconstitution function (here `RestoreOrder`) that sets fields without validating or raising events; only adapters call it:

```go
func (r *orderRepository) Get(ctx context.Context, id string) (domain.Order, error) {
	row, err := r.q.GetOrder(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	if err != nil {
		return domain.Order{}, ErrOrderLoad(err)
	}

	lines, err := r.q.ListOrderLines(ctx, id)
	if err != nil {
		return domain.Order{}, ErrOrderLoad(err)
	}

	return domain.RestoreOrder(
		row.ID,
		row.CustomerID,
		domain.ToOrderStatus(row.Status),
		rkit.Map(lines, toDomainOrderLine),
	), nil
}
```

## The Specification Pattern

The Specification pattern encapsulates query criteria as first-class domain objects. Instead of building queries in service code, you express criteria as composable specification objects.

**Go note:** do not build a generic specification framework (`Specification` interface with `And`/`Or`/`Not` combinators). In RonyKit the same intent is served by two small tools: a **domain predicate method** for in-memory checks, and a **plain filter struct** (or a named port method) for queries the database must answer.

### Why Specifications

Without specifications, query logic scatters across the codebase:

```go
// Query logic inside use cases -- not reusable, and duplicated with slight variations.
func (a *App) RiskyOrders(ctx context.Context) (*sql.Rows, error) {
	return a.db.QueryContext(ctx, "SELECT * FROM orders WHERE total > 1000000 AND customer_risk > 7")
}

func (a *App) VeryRiskyOrders(ctx context.Context) (*sql.Rows, error) {
	return a.db.QueryContext(ctx, "SELECT * FROM orders WHERE total > 5000000 AND customer_risk > 9")
}
```

With a filter struct passed to a port method:

```go
type OrderFilter struct {
	MinTotal        int64 // minor units
	MinCustomerRisk int
}

type OrderRepository interface {
	List(ctx context.Context, f OrderFilter) (domain.Orders, error)
}

func (a *App) RiskyOrders(ctx context.Context) (domain.Orders, error) {
	return a.orders.List(ctx, OrderFilter{MinTotal: 1_000_000, MinCustomerRisk: 7})
}
```

The criteria have a name and a single implementation (one sqlc query in `data/db/queries/order.sql`), and the use case speaks domain language.

### Specification Composition

Specifications compose using logical operators. With a filter struct, the composition is explicit and bounded:

| Operator | Meaning | Filter-struct equivalent |
|----------|---------|---------|
| AND | Both must be true | Set several fields: `OrderFilter{MinTotal: ..., MinCustomerRisk: ...}` |
| OR | Either must be true | A slice field (`Statuses []domain.OrderStatus`) or a separately named port method |
| NOT | Must not be true | A named field (`ExcludeCancelled bool`) -- the name states the business rule |

If a combination recurs and has a business name ("at-risk order"), give it a named port method rather than more filter fields.

### Specifications in the Domain Layer

In-memory specifications are predicate methods on the domain type; database-backed specifications are port methods. Both carry the same business name:

```go
// internal/domain: the rule itself.
func (inv Invoice) IsOverdue(asOf time.Time) bool {
	return inv.dueDate.Before(asOf) && inv.status != InvoiceStatusPaid
}

// internal/repo/port.go: the same rule, answered by the database.
type InvoiceRepository interface {
	ListOverdue(ctx context.Context, asOf time.Time) (domain.Invoices, error)
}
```

The integration test for `ListOverdue` should assert that every returned invoice satisfies `IsOverdue(asOf)`, which keeps the SQL and the domain rule from drifting apart.

## Ports and Adapters Relationship

Repositories and Factories fit naturally into the Ports and Adapters (Hexagonal) architecture:

```
                   internal/domain + internal/repo (port.go)
                   ┌──────────────────────────┐
                   │  Aggregates              │
                   │  Value Objects           │
                   │  Domain Events           │
                   │  Constructors (NewX)     │
                   │  Repository Interfaces ──┼── Port (repo/port.go)
                   └──────────────────────────┘
                              ▲
                              │ implements (bound with fx.As)
                              │
                   internal/repo/v0 and fakes
                   ┌──────────────────────────┐
                   │  orderRepository (sqlc) ─┼── Adapter (Postgres)
                   │  fakeOrderRepo          ─┼── Adapter (in-memory, app tests)
                   │  crmCustomerDirectory   ─┼── Adapter (generated stub, ACL)
                   └──────────────────────────┘
```

**The key principle:** The domain and the ports define what the app needs. Adapters provide it. Dependencies point inward -- `v0repo` depends on `repo` and `domain`, never the reverse.

This means:
- `internal/domain` has zero imports from `internal/repo/v0`, `data/db`, or `api/`
- Repository interfaces use domain types (`domain.Order`, order IDs), not persistence types (sqlc rows, `*sql.Rows`)
- The application can swap persistence technologies by providing a new adapter (a `v1` package) without touching domain or app code
- `internal/app` unit tests use in-memory fakes of the ports; the `v0` adapter is tested against real Postgres in `internal/repo/integration_test/` via `x/testkit`
