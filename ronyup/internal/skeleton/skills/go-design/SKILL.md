---
name: go-design
description: >-
  Structure and restructure Go code the Go way: dependency direction between
  packages, consumer-defined interfaces, composition instead of inheritance,
  smell-driven refactoring in small safe steps, and changing untested code with
  characterization tests and seams. Use when placing code in a RonyKit feature
  module, designing packages or interfaces, refactoring, translating a
  class-based pattern (SOLID, Strategy, Template Method, Factory, subclassing)
  into Go, or changing code that has no tests. For module depth and leaky
  abstractions, see software-design-philosophy. For domain modeling, see
  domain-driven-design. For test mechanics, see go-testing.
---

# Go Design

Go has packages, structs, functions, and implicitly satisfied interfaces — no
classes, no inheritance, no overriding. Most class-based design advice either
translates into something simpler or doesn't apply. This skill keeps the ideas
that survive the translation and states them in Go terms.

## When to use

- Deciding where code goes in a feature module, or reviewing a PR that crosses
  layers.
- Designing a package, an interface, or a constructor.
- Refactoring: a smell is slowing you down and tests are green.
- A book, blog, or habit suggests a class hierarchy and you need the Go shape.
- Changing code that has no tests (legacy code is simply code without tests).

## 1. Dependency direction

Business rules must not depend on delivery or storage details. In a RonyKit
feature module the direction is fixed:

| Package | Owns | May import |
| ------- | ---- | ---------- |
| `api/` | Rony contract, DTOs (`json` tags), request → call → response mapping | `internal/app`, `internal/domain`, `rony` |
| `internal/app` | Use cases as `*App` methods; interfaces it consumes | `internal/domain`, `internal/repo` (ports), `rony/errs` |
| `internal/domain` | Types, invariants, rules, sentinel errors | stdlib, `rony/errs`, `x/rkit` |
| `internal/repo` (`port.go`) | Repository interfaces in domain types | `internal/domain` |
| `internal/repo/v0` | sqlc implementation; row ↔ domain mapping | `internal/repo`, `internal/domain`, sqlc output |
| `module.go` / `service.go` | fx wiring: the only place concrete adapters meet | everything above |

- Data crosses boundaries as domain types or plain values — never sqlc rows,
  DTOs, or `*RContext`.
- The `rony` package (contexts, routes) stays in `api/` and wiring;
  `rony/errs` is fine everywhere.
- Other feature modules are reached through generated stubs, never by
  importing their `internal/`.
- Keep the handler humble: decode, call one `App` method, map the result. What
  is hard to test (transport) stays thin; what matters (rules) is testable
  without it.

Check the direction mechanically:

```bash
go list -f '{{.ImportPath}}: {{join .Imports " "}}' ./internal/app/... ./internal/domain/... \
  | rg -e '/api( |$)' -e '/repo/v0' -e 'ronykit/rony( |$)'
```

Any hit is a violation. RonyKit specifics: MCP
`architecture/service-structure`, `architecture/domain-layer`,
`architecture/repo-ports`, `architecture/module-wiring`.

## 2. Interfaces the Go way

- **Defined by the consumer, not the implementer.** `internal/app` declares
  what it needs; `v0repo` or an SDK adapter satisfies it without importing the
  interface's package in its type declaration.
- **Small.** One to a handful of methods, named for the domain
  (`OrderRepository`, `PaymentGateway`), not for the technology.
- **Accept interfaces, return concrete types.** Constructors return `*T`, not
  an interface, unless the wiring needs `fx.As`.
- **No interface without a reason.** Reasons: a boundary (repo port, external
  service), or a second real implementation. "Maybe we'll swap it" and "to mock
  it" for code you own are not reasons — test through the real type.
- **Fakes must honor the contract.** If the real repo returns
  `domain.ErrOrderConflict` on a duplicate, the in-memory fake must too;
  otherwise app tests pass against a lie.

```go
// internal/app/payment.go
type PaymentGateway interface {
	Charge(ctx context.Context, orderID string, amount int64) (chargeID string, err error)
}

// The adapter lives outside internal/app and is bound in module.go with
// fx.Annotate(NewGateway, fx.As(new(app.PaymentGateway))).
type Gateway struct{ client *stripe.Client }

var _ app.PaymentGateway = (*Gateway)(nil) // compile-time contract check
```

## 3. Composition instead of inheritance

Translate the pattern, not the class diagram:

| Class-based habit | Go shape |
| ----------------- | -------- |
| Base class sharing behavior | A plain function, or a helper type held as a named field |
| Subclass overrides a step (Template Method) | Pass the varying step as a `func` parameter or field |
| Strategy class hierarchy | A `func` type or a one-method interface |
| Replace Conditional with Polymorphism | Closed set of cases: `switch` on a typed constant with a `default` that returns an error. Open, pluggable set: an interface |
| Decorator / proxy | A struct that holds the interface and implements it (middleware style) |
| Factory class | `NewX(...) (X, error)` that enforces invariants |
| Singleton | One instance built by fx and injected; no mutable package globals |
| Getters/setters for every field | Unexported fields; getter named `Status()` (not `GetStatus()`); methods named for operations (`Cancel(now)`), not `SetStatus` |
| Abstract class / "interface + abstract impl" | An interface and independent implementations; share code through functions |

Embedding is promotion, not "is-a": every exported method of the embedded type
becomes part of your API, and a nil embedded interface panics on the first
call it doesn't override. Embed deliberately; otherwise use a named field (for
example `mu sync.Mutex`, never an embedded mutex in an exported type, which
would expose `Lock`/`Unlock`).

In TypeScript/React the same rule holds: discriminated unions with an
exhaustive `switch` (a `never` check in `default`) instead of class
hierarchies, and component composition instead of inheritance (see
`composition-patterns`).

## 4. Packages

- Name a package for what it provides (`invoice`, `ledger`, `stripe`), never
  `util`, `common`, `helpers`, `models`, or `types`.
- Prefer fewer, deeper packages. Splitting files inside a package is free;
  splitting a package adds an API surface and import edges.
- Go rejects import cycles. When one appears, the missing concept usually
  belongs in the inner package (often `domain`), or the consumer should own a
  small interface — don't create a `shared` package to hide the cycle.
- `internal/` enforces boundaries the compiler can check; use it instead of
  comments that say "do not import".
- Group by what changes together: one feature's domain, use cases, and SQL
  live in one feature module, not split by technical layer across modules.

## 5. Refactoring in small, safe steps

Refactoring changes structure, never behavior. The workflow:

1. Tests are green before you start. If there are none, go to section 6.
2. Make one named move, compile, run the focused tests. Repeat.
3. Commit structure changes separately from behavior changes; a reviewer (and
   `git bisect`) can then trust the structure-only commits.
4. Use tooling for mechanical moves: `gopls` rename and extract (editor code
   actions), `gofmt -r` for expression rewrites.
5. Never hand-edit generated code (sqlc output, stubs). Change `query.sql` or
   the contract, then `make sqlc` / `make gen-stub`.

Smells and their Go remedies:

| Smell | Remedy |
| ----- | ------ |
| Long function mixing abstraction levels | Extract functions named for intent; judge by levels, not line count (Go error handling is verbose) |
| Deep nesting | Guard clauses and early return; happy path at the left margin |
| Boolean flag parameter | Two functions, or a typed option |
| Long parameter list / data clump | A params struct; functional options only for optional configuration |
| Primitive obsession | Named types with methods: `type OrderID string`, `domain.Money`, validated by a constructor |
| Feature envy (handler or `App` poking at a type's fields) | Move the logic onto the type, usually in `internal/domain` |
| The same `switch` repeated in several places | One method on the type, or an interface if the set is open |
| Shotgun surgery (one change touches many packages) | Gather the knowledge into one package |
| Huge `app.go` | Split `App` methods into files per concept (same package); push rules into domain types |
| Comment explaining *what* a block does | Extract a function whose name says it |
| Interface with one implementation and no boundary | Inline it; use the concrete type |
| Dead or speculative code | Delete it; git remembers |

## 6. Changing code that has no tests

Cover, then modify. Never edit and pray.

1. **Find the change point** and the nearest **test point** — often a public
   method whose result your change affects.
2. **Break only the dependency that blocks the test**, with the least invasive
   move (below). Preserve signatures and lean on the compiler.
3. **Write characterization tests**: assert something absurd, run, read the
   failure, pin the observed value as a literal. The current behavior is the
   spec, quirks included.
4. **Make the change** inside that net, then refactor.

```go
func TestLateFee_Characterization(t *testing.T) {
	due := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Pinned from the first failing run; do not recompute here.
	assert.Equal(t, int64(1500), lateFee(10_000, due, due.AddDate(0, 0, 31)))
	assert.Equal(t, int64(0), lateFee(10_000, due, due.AddDate(0, 0, 30))) // BUG? expected fee on day 30 — TICKET-482
}
```

Found a bug while characterizing? Pin it with a comment and a ticket, then fix
it in a separate, deliberate commit — callers may depend on it. For large
outputs, use golden files (`testdata/*.golden` behind an `-update` flag) with
IDs and timestamps normalized.

**Seams available in Go**, cheapest first:

| Blocker | Move |
| ------- | ---- |
| Constructor opens a DB/HTTP client itself | Parameterize the constructor (in RonyKit, add it to the `fx.In` params) |
| Concrete dependency you can't construct in a test | Declare a small interface at the consumer; the concrete type needs no edit |
| Wall clock, random, or ID generation | A `now func() time.Time` / `newID func() string` field set by the constructor; avoid mutable package vars (they break `t.Parallel()`) |
| Package-level function used everywhere (`billing.Charge()`) | Move behavior onto a struct; keep the package function delegating to a default instance until callers migrate |
| Logic buried in a handler taking `*RContext` | Extract it into a function or `App` method taking plain values |
| 500-line function hoarding locals | Extract a small struct whose fields are those locals, with a `run()` method, then refactor inside it |
| Nothing else works | Build tags (`//go:build integration`) — the bluntest seam |

**When the host can't be tested today:**

- *Sprout*: write the new behavior as a new, test-first function or type and
  call it from one line in the untested code.
- *Wrap*: put behavior around a call with a decorator that implements the
  same interface:

```go
type auditedGateway struct {
	next PaymentGateway
	log  *logkit.Logger
}

func (g auditedGateway) Charge(ctx context.Context, orderID string, amount int64) (string, error) {
	chargeID, err := g.next.Charge(ctx, orderID, amount)
	g.log.InfoCtx(ctx, "charge attempted", logkit.String("order_id", orderID), logkit.Bool("ok", err == nil))

	return chargeID, err
}
```

Sprouts leave the host untested: cover it the next time a change lands there.
Then bring the feature to workspace standard (repo integration tests, `App`
unit tests — see `go-testing`).

## Common mistakes

| Mistake | Why it fails | Fix |
| ------- | ------------ | --- |
| Porting a class hierarchy (base struct + "overrides" via embedding) | Embedding doesn't dispatch virtually; the "base" calls its own methods, never yours | Functions, `func` fields, or an interface |
| Interface per struct, declared next to it | Producer-side abstraction with no second implementation | Declare at the consumer, only at boundaries |
| Per-use-case interactor structs and boundary interfaces | Java ceremony; indirection with no payoff | One `*App` with methods; interfaces only for what `App` consumes |
| sqlc rows or DTOs in `internal/app` | Schema or wire changes rewrite business code | Map at the `v0repo` / `api` edge |
| Refactoring and changing behavior in one commit | A failure can't be attributed | Separate commits; tests green between them |
| "Should" tests written against legacy code | Imagined specs; you "fix" behavior callers rely on | Characterize what it does; fix bugs separately |
| Mocking everything | Tests pin the implementation and break on every refactor | Fake only what blocks construction or sensing |

## Checklist before "done"

- Imports point inward; the `go list` check above prints nothing.
- Every new interface sits at its consumer and has a boundary or a second
  implementation behind it.
- No class-hierarchy emulation; embedding is deliberate.
- Structure and behavior changes are in separate commits, tests green after
  each.
- Changed code that lacked tests now has characterization tests pinning
  observed values.
- `make lint` and `go test ./... -race` pass in every touched module.

## Attribution

Ideas distilled from Robert C. Martin's *Clean Architecture*, Martin Fowler's
*Refactoring*, and Michael Feathers' *Working Effectively with Legacy Code*,
by way of [`wondelai/skills`](https://github.com/wondelai/skills) (MIT), and
rewritten for Go and RonyKit feature modules.
