Every service must have an `internal/domain` package.

- Create `domain/doc.go` with just the package declaration.
- Define domain types as plain Go structs without `json` tags
  (`json` serialization belongs in the API layer).

Use type-safe enum patterns:

- define a struct type with an unexported `slug` field,
- add a `String()` method,
- expose package-level variables for each valid value
  (for example `AccountTypeUser = AccountType{slug: "user"}`).

Provide a `ToXxx(string)` constructor that maps strings to enum values with a
default/unknown fallback.

For collection types, define named slice types (for example
`type Ledgers []Ledger`) with helper methods like `GetByID`, `GetByName`, `Len`.

Use input types with unexported fields and getter methods when constructors need
validation (for example `NewLedgerInput` validates scale before returning).

Place all domain errors in `domain/errors.go`.
