---
name: code-review
description: >-
  Review code (your own or others') for correctness, design, and risk before it
  merges. Use when reviewing a diff or pull request, doing a self-review before
  opening a PR, or giving structured, prioritized feedback.
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
   package-selection rules; no leaking concerns across boundaries.
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
- **Scale is actual, not imagined** — generality/caching/sharding for traffic
  you don't have is speculation; prefer the simplest thing that fits today.
- **Dependencies earn their keep** — a new library must save more than its
  footprint and upgrade burden; prefer a workspace helper (`x/*`, `rony/*`).
- **Premature optimization** — no micro-optimizations without a measurement;
  readable beats clever on cold paths.

Calibrate: a junior over-abstracting is learning; flag the pattern, propose the
simpler alternative, and keep the person's dignity. Distinguish this from
genuine, requirement-driven complexity — the point is fit, not minimalism for
its own sake.

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

## Red flags

- Large diff with no tests, or tests that can't fail.
- Broad `try/catch`/`recover` swallowing errors.
- New dependency where a workspace helper exists.
- Commented-out code or unexplained magic numbers.
- An abstraction/config layer with a single implementation, or generality built
  for scale the system doesn't have yet.
