---
name: go-modern
description: >-
  Write idiomatic, modern Go (1.25+/1.26) using current language and stdlib
  features and the RonyKIT package map. Use when authoring or reviewing Go
  code, choosing between stdlib, `x/*`, and third-party helpers, or working
  with errors (`rony/errs`), generics, iterators, contexts, goroutines, or doc
  comments in this workspace. For tests, see go-testing. For package and
  interface design, layer placement, and refactoring, see go-design.
---

# Modern Go

Write Go that reads like the standard library: small, explicit, and boring in
the best way. Target the workspace Go version (1.25+/1.26).

## When to use

- Implementing or refactoring Go packages, services, or CLIs.
- Reviewing Go for idiomatic style and current stdlib usage.
- Deciding between stdlib, a RonyKIT `x/*` helper, and a third-party package.

## Core idioms

- **Accept interfaces, return structs.** Keep interfaces small and defined by
  the consumer (for example `internal/repo/port.go` is defined for `app`, not
  by the sqlc adapter).
- **Names reveal intent.** Prefer `elapsedDays` over `d`; booleans as
  predicates (`isActive`, `hasPermission`); functions as verb+noun. Package
  names are short and singular; feature packages are `<feature>mod`. Don't
  stutter (`invoice.Invoice` is fine, `invoice.InvoiceService` is not).
- **Context first.** `ctx context.Context` is the first parameter; never store
  it in a struct. Honor cancellation and deadlines; pass it to every I/O call.
- **Zero values are useful.** Design types so the zero value is ready to use,
  or make construction go through `NewX(...) (X, error)` when invariants apply
  (see `domain-driven-design`).
- **Generics for containers/algorithms only.** Don't reach for type parameters
  when a concrete type or a small interface is clearer.
- **Early return.** Handle errors and edge cases first; keep the happy path at
  the left margin.

## Errors

Service code uses `rony/errs` only — never `errors.New` / `fmt.Errorf` for
domain or API errors. Define sentinels in `internal/domain/errors.go`:

```go
// internal/domain/errors.go
var (
	ErrInvoiceInvalid = errs.B().Code(errs.InvalidArgument).Msg("INVOICE_INVALID").Err()
	ErrInvoiceSave    = errs.GenWrap(errs.Internal, "INVOICE_SAVE_FAILED")
)

// internal/app
func (a *App) SaveInvoice(ctx context.Context, inv domain.Invoice) error {
	if err := a.invoices.Save(ctx, inv); err != nil {
		return domain.ErrInvoiceSave(err) // GenWrap(nil) returns nil — never use it for validation
	}

	return nil
}
```

- Inspect with `errors.Is` / `errors.As`; combine independent failures with
  `errors.Join`.
- Handle an error once: return it **or** log it, not both.
- Codes are `SCREAMING_SNAKE_CASE`; pick the `errs` code that maps to the right
  HTTP status (`NotFound`, `AlreadyExists`, `FailedPrecondition`, …). Read
  `knowledge://ronyup/packages/errs`.

## Use current stdlib

- Iterators (`iter.Seq`, `range`-over-func). For map/filter/reduce/paginate in
  service code prefer `x/rkit` (`rkit.Map`, `Filter`, `Reduce`, `Paginate`);
  `slices` / `maps` / `cmp` are fine when no RonyKIT helper fits.
- `min`/`max`/`clear` builtins; `for i := range n` for counted loops.
- Loop variables are per-iteration since Go 1.22 — don't add `v := v` copies.
- `context.WithoutCancel`, `WithDeadlineCause`, and `AfterFunc` where they fit.
- `t.Context()` in tests (Go 1.24+) instead of `context.Background()`.
- Structured logging: `x/telemetry/logkit` only — never `log`, `log/slog`, or
  `zap`.

## Package selection (this workspace)

Before importing a third-party or stdlib helper, check for a RonyKIT
equivalent: IDs/JSON-byte casts/string↔number/case/collections → `x/rkit`;
config → `x/settings`; DI → `x/di`; errors → `rony/errs`; logging →
`x/telemetry/logkit`; pools → `x/p`; micro-batching → `x/batch`. Durable
workflows: `flow` only — never import `go.temporal.io/sdk` directly. See
`knowledge://ronyup/architecture/package-selection`.

## Concurrency

- Start a goroutine only when you control its lifetime and shutdown; every
  goroutine needs a way to stop (context, closed channel, or `WaitGroup`).
- Prefer `errgroup` / bounded worker pools over unbounded `go` calls.
- Protect shared state with the smallest possible critical section; prefer
  channels for ownership transfer, mutexes for protecting fields.
- Don't start goroutines in request handlers that outlive the request; use a
  `flow` workflow for durable background work.
- Always test concurrent code with `-race`.

## Common mistakes

| Mistake | Why it fails | Fix |
| ------- | ------------ | --- |
| `fmt.Errorf("save: %w", err)` in service code | Loses the `errs` code → wrong HTTP status, inconsistent API errors | `ErrXSave(err)` via `errs.GenWrap` |
| Logging and returning the same error | Duplicate log lines at every layer | Return it; log once at the edge |
| `context.Background()` inside a request path | Drops deadlines, cancellation, and trace context | Thread the caller's `ctx` |
| Interface declared next to its only implementation | Producer-side abstraction; couples callers to it | Declare it where it is consumed, or drop it |
| `defer` inside a long loop | Resources held until the function returns | Extract the loop body into a function |
| Returning `nil` slice vs empty slice inconsistently in API output | JSON `null` vs `[]` breaks clients | Normalize at the handler / DTO boundary |
| Hand-rolled `uuid`, `strconv`, JSON helpers | Fails `depguard`; bypasses shared behavior | `x/rkit` equivalents |

## Checklist before "done"

- `make lint` clean (formatters + `depguard`), `go vet` passes.
- No naked `panic` in library code; errors returned via `rony/errs`.
- Every goroutine has an owner and a stop condition.
- Exported identifiers have doc comments starting with the identifier name
  that state the contract, not the implementation.
