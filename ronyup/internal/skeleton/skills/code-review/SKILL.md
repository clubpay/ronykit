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
