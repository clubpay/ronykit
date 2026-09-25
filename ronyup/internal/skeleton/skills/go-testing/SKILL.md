---
name: go-testing
description: >-
  Write clear, fast, deterministic Go tests including mandatory repo integration
  tests (x/testkit) and app unit tests. Use when adding or fixing Go tests,
  implementing internal/repo or internal/app, designing table-driven tests,
  writing benchmarks, using testify, or improving coverage in this workspace.
  For language-agnostic TDD and what deserves a test, see writing-tests. For
  untested code you must change, see go-design first.
---

# Go Testing

Tests are documentation that runs. Make them readable, deterministic, and fast.

## Repo integration tests (mandatory — enforced by verify.sh)

Every method on every repository port in `internal/repo/port.go` **must** have an integration test in
`internal/repo/integration_test/` using `x/testkit` against real Gnomock Postgres/Redis — not mocks.

Per method, cover:

- **Happy path** — creates/reads/updates as intended
- **Not found** — missing row / empty result
- **Conflict** — unique violation or constraint error

Read MCP `architecture/integration-tests`. Scaffold provides `setup_test.go`; add `*_test.go` files per domain. **Run**
`go test ./internal/repo/integration_test/...` and confirm green before treating the repo as done.

## App unit tests (mandatory — enforced by verify.sh)

Every exported method on `internal/app.App` needs a unit test in `internal/app/*_test.go`. Test business rules with injected fakes/mocks
at the port boundary — not integration tests.

- Prefer a small **in-memory fake** of the repo port (a map behind the interface) over call-recording mocks: assert on the resulting
  state and the returned value, not on which port methods were called.
- Cover each business rule and each error path (`errors.Is(err, domain.ErrX)`), not just the happy path.
- Inject clocks and ID generators through `App` dependencies so expected values are fixed, hand-written constants.

## When to use

- Adding tests for new Go behavior or covering a bug fix.
- Designing table-driven tests, subtests, fuzz tests, or benchmarks.
- Diagnosing flaky, slow, or order-dependent tests.

## Conventions

- **Table-driven + subtests.** One `t.Run(tc.name, ...)` per case so failures
  point at the exact scenario.
- **Arrange / Act / Assert.** Keep the three sections visually distinct.
- **Deterministic.** No real clocks, randomness, network, or sleeps. Inject
  time/IDs; use `context.Context` with deadlines.
- **Parallel where safe.** `t.Parallel()` for independent cases (loop
  variables are per-iteration since Go 1.22 — no `tc := tc`). Don't
  parallelize tests that share a database schema or global state.
- **Helpers call `t.Helper()`** so failures report the caller's line.
- **`t.Context()`** (Go 1.24+) for contexts that end with the test;
  `t.Cleanup` over `defer` in helpers.
- **Golden files** for large outputs; gate updates behind a `-update` flag and
  review the diff before committing.

## No tautological tests

A test that restates the implementation proves nothing: it passes for any code
that compiles and breaks on every refactor. Delete these on sight rather than
fixing them:

- Asserting a mock returns what the test told it to return.
- Computing `want` with the same expression or helper the code under test uses
  — write the expected value by hand.
- `assert.Equal(t, Const, Const)` or asserting on a struct you just built.
- Asserting only `assert.NoError` without checking the result or side effect.

A characterization test that pins an **observed** value (see `go-design`) is
fine; one that recomputes it at assert time is not.

## Framework

Use standard **`testing`** with **`github.com/stretchr/testify`** (`assert` /
`require`). Prefer table-driven tests and `t.Run` subtests. Match surrounding
package style; do not introduce alternate assertion libraries.

```go
import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    int
		wantErr bool
	}{
		{name: "valid", in: "42", want: 42},
		{name: "empty", in: "", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Parse(tc.in)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
```

## Benchmarks & fuzzing

- `func BenchmarkX(b *testing.B)` with `b.ResetTimer()` after setup and
  `b.ReportAllocs()` when allocations matter.
- Add a `func FuzzX(f *testing.F)` with seed corpus for parsers and decoders.

## Commands

```bash
cd <module>
go test ./... -race                 # always run with the race detector
go test ./... -run TestName -v      # focus one test
go test ./... -bench . -benchmem    # benchmarks
```

## Common mistakes

| Mistake | Why it fails | Fix |
| ------- | ------------ | --- |
| Mocking the database in repo tests | SQL, constraints, and sqlc mapping go untested | `x/testkit` + Gnomock Postgres in `integration_test/` |
| `assert.NoError` where later lines depend on success | Test continues and panics or reports noise | `require.NoError` for preconditions, `assert` for checks |
| `time.Sleep` to wait for async work | Slow and flaky | Poll with `assert.Eventually` or inject a synchronization point |
| One giant test per method | First failure hides the rest; unclear cause | Table-driven cases with `t.Run` names that read as specs |
| Only the happy path | Error mapping (`errs` codes) regresses silently | Add not-found / conflict / invalid cases |

## Checklist

- Test names describe behavior, not implementation.
- Failures print expected vs got with enough context to debug.
- New behavior and every fixed bug has a covering test.
- Every repo port method has a passing `x/testkit` integration test.
- Every exported `App` method has a unit test.
- `make verify` passes before reporting backend work done.
