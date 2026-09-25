---
name: writing-tests
description: >-
  Language-agnostic discipline for writing valuable tests and practicing TDD.
  Use when deciding what to test, structuring a test suite, doing red-green-
  refactor, or judging whether coverage is meaningful rather than vanity. For
  Go specifics and the mandatory repo/app tests, see go-testing; for
  Vitest/Playwright, see frontend-testing; for Go code with no tests yet, see
  go-design.
---

# Writing Tests

Test behavior, not implementation. A good test fails only when the contract
breaks, and tells you exactly what broke.

## When to use

- Before/while writing new behavior (TDD), or backfilling tests for legacy code
  (pin current behavior with characterization tests first; for Go, see
  `go-design`).
- Deciding what deserves a test and at which level.
- A bug was found — write the failing test first, then fix it.

## The test pyramid

1. **Unit** (most): pure logic, fast, no I/O. Milliseconds.
2. **Integration** (some): real boundaries — DB, queue, HTTP — wired together.
3. **End-to-end** (few): full system through the public interface.

Push assertions down to the cheapest level that still proves the behavior.

## TDD loop

1. **Red** — write the smallest failing test for the next behavior.
2. **Green** — write the least code to pass it.
3. **Refactor** — clean up with tests green.

Commit at green. Never refactor and change behavior in the same step.

## What makes a test valuable

- **Tests behavior, not internals.** Assert on observable outputs and effects,
  not private fields or call counts (avoid over-mocking).
- **One reason to fail.** Each test pins one behavior; the name says which.
- **Deterministic.** No reliance on time, ordering, network, or shared state.
- **Readable.** A reviewer understands the contract from the test alone.
- **Fast.** Slow suites get skipped; isolate slow tests behind tags/build flags.

## Coverage

Coverage finds untested code; it does not prove correctness. Chase missing
branches and error paths, not a percentage. A line covered without an assertion
is not tested.

## Anti-patterns

- **Tautological tests** — they restate the implementation instead of pinning
  behavior: asserting a mock returns what you configured, re-deriving the
  expected value with the same expression the code uses, `expect(X).toBe(X)`,
  or approving a snapshot nobody read. Use hand-computed expected values;
  delete tautological tests rather than fixing them.
- Asserting implementation details that change during refactors.
- Tests that pass when the feature is broken (no real assertion).
- Mocking the thing under test, or mocking value objects.
- Shared mutable fixtures that couple unrelated tests.
- Snapshot tests so large nobody reviews the diff.

## Checklist

- Each new behavior and each fixed bug has a test that fails without the change.
- Test names read as specifications.
- Suite is deterministic and runnable in isolation and in any order.
