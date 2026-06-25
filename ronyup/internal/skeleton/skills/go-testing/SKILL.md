---
name: go-testing
description: >-
  Write clear, fast, deterministic Go tests including mandatory repo integration
  tests (x/testkit) and app unit tests. Use when adding or fixing Go tests,
  implementing internal/repo or internal/app, designing table-driven tests,
  writing benchmarks, using testify/Ginkgo, or improving coverage in this workspace.
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
- **Parallel where safe.** `t.Parallel()` for independent cases; capture loop
  variables (not needed in Go 1.22+, but be explicit if unsure).
- **Helpers call `t.Helper()`** so failures report the caller's line.
- **Golden files** for large outputs; gate updates behind a `-update` flag.

## Framework

This repo uses **Ginkgo v2 + Gomega** where BDD style is established; plain
`testing` + table tests elsewhere. Match the surrounding package's style — do
not introduce a new framework into a package that already has one.

```go
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
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q) = %d, want %d", tc.in, got, tc.want)
			}
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

## Checklist

- Test names describe behavior, not implementation.
- Failures print expected vs got with enough context to debug.
- New behavior and every fixed bug has a covering test.
- Every repo port method has a passing `x/testkit` integration test.
- Every exported `App` method has a unit test.
- `make verify` passes before reporting backend work done.
