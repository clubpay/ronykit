Every service must have an `internal/domain` package.

- Create `domain/doc.go` with just the package declaration.
- Define domain types as plain Go structs without `json` tags (`json` serialization belongs in the API layer).

## Model rich domain entities — avoid anemic structs

Do NOT model the domain as bags of public fields whose invariants are enforced elsewhere (handlers, app, repo). Put the behavior and the
rules that protect invariants on the domain types themselves.

- **Entities** have identity and a lifecycle. Give them methods that express domain behavior and mutate state through guarded operations
  (for example `account.Debit(amount)` returns an error instead of letting callers set a balance field directly). Keep fields that back an
  invariant unexported and expose getters/behavior, so illegal states are unrepresentable from outside the package.
- **Value objects** are immutable, identity-less, and compared by value (for example `Money`, `Email`, `Scale`, `Currency`). Construct them
  through validating constructors (`NewMoney(...) (Money, error)`) that reject invalid input, then treat them as read-only. Prefer a value
  object over a bare primitive whenever the value has rules (range, format, units).
- **Aggregates** are a consistency boundary: one root entity owns its child entities/value objects, and all external access goes through the
  root. Outside callers reference an aggregate only by its root identity. Enforce cross-entity invariants inside the aggregate root's methods
  so the aggregate is always valid as a whole.

Guidance:

- Put behavior on the entity/aggregate when the rule belongs to a single concept; keep multi-aggregate orchestration and infrastructure
  concerns in `internal/app`. The app layer coordinates aggregates and persistence; it does not re-implement invariants the domain already
  guarantees.
- Use constructors / factory functions that return `(T, error)` so an instance cannot exist in an invalid state.

Use type-safe enum patterns:

- define a struct type with an unexported `slug` field,
- add a `String()` method,
- expose package-level variables for each valid value (for example `AccountTypeUser = AccountType{slug: "user"}`).

Provide a `ToXxx(string)` constructor that maps strings to enum values with a default/unknown fallback.

For collection types, define named slice types (for example `type Ledgers []Ledger`) with helper methods like `GetByID`, `GetByName`, `Len`.

Use input types with unexported fields and getter methods when constructors need validation (for example `NewLedgerInput` validates scale before returning).

Place all domain errors in `domain/errors.go`.
