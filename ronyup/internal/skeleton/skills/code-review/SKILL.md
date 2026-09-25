---
name: code-review
description: >-
  Review code (your own or others') for correctness, design, and risk before it
  merges. Use when reviewing a diff or pull request, doing a self-review before
  opening a PR, or giving structured, prioritized feedback. For layer
  violations, smell fixes, and untested changes in Go, see go-design; for
  shallow or leaky abstractions, see software-design-philosophy.
---

# Code Review

Review for correctness and design first, style last (a formatter owns style).
Prioritize feedback so the important things aren't lost in nits.

## When to use

- Reviewing a teammate's PR or diff.
- Self-reviewing your own change before requesting review.
- Deciding whether a change is safe to merge.

## What to check, in priority order

1. **Correctness** — does it do what it claims? Edge cases, error paths,
   off-by-one, nil/empty, concurrency, boundary conditions.
2. **Security & safety** — input validation, authz checks, injection, secrets in
   code/logs, unsafe defaults.
3. **Design fit** — right layer/abstraction; follows existing architecture and
   package-selection rules; no leaking concerns across boundaries. For RonyKit
   features, read `go-design` — handlers thin, logic in `internal/app`,
   persistence behind `internal/repo/port.go`, imports pointing inward.
4. **Tests** — meaningful tests for new behavior and fixed bugs; deterministic;
   assert behavior not internals.
5. **Readability** — clear names, small functions, comments only where intent is
   non-obvious.
6. **Style** — defer to the formatter/linter; don't hand-review spacing.

## Watch for over-engineering

Complexity is a cost, not a credential. Flag code built to impress (or "just in
case") rather than to meet a real requirement — it compounds maintenance with
no user value. Quick lenses:

- **Deletion test** — if this were removed, who'd notice and when? "Only the
  author" means it's not pulling its weight.
- **Abstraction needs ≥3 uses** — one implementation behind an interface is
  indirection, not abstraction. Don't add a layer for a hypothetical second case.
  (Architectural boundaries such as `internal/repo/port.go` are the exception:
  they exist to keep the app testable and storage-independent.)
- **Scale is actual, not imagined** — generality/caching/sharding for traffic
  you don't have is speculation; prefer the simplest thing that fits today.
- **Dependencies earn their keep** — a new library must save more than its
  footprint and upgrade burden; prefer a workspace helper (`x/*`, `rony/*`).
- **Premature optimization** — no micro-optimizations without a measurement;
  readable beats clever on cold paths.

Calibrate: a junior over-abstracting is learning; flag the pattern, propose the
simpler alternative, and keep the person's dignity. Distinguish this from
genuine, requirement-driven complexity — the point is fit, not minimalism for
its own sake. For the deeper vocabulary (shallow modules, information leakage,
pass-through methods), read `software-design-philosophy`.

## How to give feedback

- **Prioritize:** label comments `blocking`, `should`, or `nit` so intent is
  clear.
- **Be specific and kind:** explain the why and suggest a concrete alternative.
- **Ask, don't assert,** when you might be missing context.
- **Praise good choices** — reinforce patterns worth repeating.
- Keep scope honest: don't demand unrelated refactors in the PR.

## Self-review before opening a PR

- Re-read your own diff top to bottom as if it were someone else's.
- Remove debug logs, dead code, and TODOs you can resolve now.
- Confirm tests, formatter, and linter pass locally.
- Write a description that states the why, the approach, and how to verify.

## RonyKit feature checklist

For a full architecture pass, run the MCP prompt `review-architecture`. At
minimum, check these in any diff touching `feature/<name>/`:

| Check | How to verify | If it fails |
| ----- | ------------- | ----------- |
| Handlers thin; rules in `internal/app` / `internal/domain` | Read `api/api_*.go`: decode → one app call → encode | `blocking` — move the logic inward |
| No forbidden imports | `rg -e go.temporal.io/sdk -e log/slog -e go.uber.org/zap -e google/uuid -g '!*_test.go'` | `blocking` — `make lint` (depguard) will fail |
| Errors via `rony/errs` | No `errors.New` / `fmt.Errorf` in service code | `should` |
| Every new port method has a repo integration test | `internal/repo/integration_test/` covers happy, not-found, conflict | `blocking` — `make verify` fails |
| Every new exported `App` method has a unit test | `internal/app/*_test.go` | `blocking` — `make verify` fails |
| Contract changed → stubs regenerated | Diff includes `stub/` changes after `make gen-stub` | `blocking` for consumers |
| SQL changed → sqlc regenerated | `data/db` generated code updated with `.sql` | `blocking` |
| Behavior matches the approved SDD | Compare with `docs/design/<feature>-sdd.md` | Update the SDD first, then the code |

## Red flags

- Large diff with no tests, or tests that can't fail.
- Broad `try/catch`/`recover` swallowing errors.
- New dependency where a workspace helper exists.
- Commented-out code or unexplained magic numbers.
- An abstraction/config layer with a single implementation, or generality built
  for scale the system doesn't have yet.

For smells and their Go remedies (feature envy, primitive obsession, flag
parameters, etc.), read `go-design`. For untested code being changed, require
characterization tests first (`go-design`, section 6).
