---
name: go-modern
description: >-
  Write idiomatic, modern Go (1.25+/1.26) using current language and stdlib
  features. Use when authoring or reviewing Go code, choosing between stdlib and
  third-party helpers, working with generics, iterators, contexts, or error
  wrapping in this workspace.
---

# Modern Go

Write Go that reads like the standard library: small, explicit, and boring in
the best way. Target the workspace Go version (1.25+/1.26).

## When to use

- Implementing or refactoring Go packages, services, or CLIs.
- Reviewing Go for idiomatic style and current stdlib usage.
- Deciding between stdlib, a RonyKIT `x/*` helper, and a third-party package.

## Core idioms

- **Accept interfaces, return structs.** Keep interfaces small and defined by the
  consumer, not the producer.
- **Errors:** wrap with `fmt.Errorf("...: %w", err)`; inspect with `errors.Is` /
  `errors.As`. In this workspace prefer `rony/errs` for domain errors.
- **Context first.** `ctx context.Context` is the first parameter; never store it
  in a struct. Honor cancellation and deadlines.
- **Zero values are useful.** Design types so the zero value is ready to use.
- **Generics for containers/algorithms only.** Don't reach for type parameters
  when a concrete type or a small interface is clearer.

## Use current stdlib

- Iterators (`iter.Seq`, `range`-over-func), `slices`, `maps`, `cmp` for
  collection work instead of hand-rolled loops.
- `min`/`max`/`clear` builtins.
- `for i := range n` for counted loops.
- `errors.Join` to combine multiple failures.
- `context.WithoutCancel`, `WithDeadlineCause`, and `AfterFunc` where they fit.
- `log/slog` for structured logging (or the workspace `x/telemetry/logkit`).

## Package selection (this workspace)

Before importing a third-party or stdlib helper, check for a RonyKIT
equivalent: IDs/JSON-byte casts/string↔number/case/collections → `x/rkit`;
config → `x/settings`; DI → `x/di`; errors → `rony/errs`; logging →
`x/telemetry/logkit`. Durable workflows: `flow` only — never import
`go.temporal.io/sdk` directly. See `knowledge://ronyup/architecture/package-selection`.

## Concurrency

- Start a goroutine only when you control its lifetime and shutdown.
- Prefer `errgroup` / bounded worker pools over unbounded `go` calls.
- Protect shared state with the smallest possible critical section; prefer
  channels for ownership transfer, mutexes for protecting fields.
- Always test with `-race` for concurrent code.

## Checklist before "done"

- `gofmt`/`goimports` clean, `go vet` passes, `golangci-lint` clean.
- No naked `panic` in library code; errors returned and wrapped.
- Exported identifiers have doc comments starting with the identifier name.
