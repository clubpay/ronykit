---
name: systematic-debugging
description: >-
  Find the root cause before changing code. Use when encountering any bug, test
  failure, crash, flaky test, performance regression, or unexpected behavior,
  before proposing or applying a fix. After the fix, use
  verification-before-completion to prove it; for production failure modes
  (timeouts, cascades), see release-it.
---

# Systematic Debugging

Guessing creates new bugs and hides old ones. Find the root cause first.

## The iron rule

> No fix without a root-cause hypothesis you have confirmed with evidence.

If you cannot explain *why* the bug happens, you are not ready to fix it.

## When to use

Any test failure, crash, wrong output, flaky test, performance regression, build
failure, or integration issue — especially when under time pressure or when a
previous fix didn't hold.

## Phase 1 — Understand

- Read the full error and stack trace; note file, line, and exact message.
- Reproduce it reliably. A bug you can't reproduce, you can't verify fixed.
- Establish the smallest reproduction and the expected vs actual behavior.

## Phase 2 — Investigate

- Form a hypothesis about the cause and predict what you'd observe if true.
- Gather evidence: targeted logging, a debugger, `git bisect`, binary search by
  commenting/disabling, or diffing a working vs broken state.
- Follow the data flow backward from the symptom to its origin. Question
  assumptions ("this can't be nil" — prove it).

## Phase 3 — Fix the cause

- Change the root cause, not the symptom. Patching where it crashed instead of
  where the bad value originated just moves the bug.
- Make the smallest change that addresses the cause.

## Phase 4 — Verify & prevent

- Write a test that fails before the fix and passes after.
- Re-run the full relevant suite to check for regressions.
- Ask whether the same class of bug exists elsewhere.

## Tactics

- `git bisect` to locate the introducing commit.
- Binary search the input/code path to isolate the trigger.
- Add structured logging at decision points; remove it before committing.
- For flakiness: suspect time, ordering, concurrency, and shared state; run with
  `-race` and `-count=20 -run '^TestName$'`.
- In a RonyKit workspace, check the cheap mechanical causes first: stale sqlc
  code (`make sqlc`), stale stubs after a contract change (`make gen-stub`),
  missing settings/env for `x/settings`, and fx wiring errors printed at
  startup (a missing provider names the type it could not build).
- Use `x/telemetry` traces and logkit fields (request/trace IDs) to follow a
  request across services instead of adding ad-hoc prints.

## Anti-patterns

- Trying random changes to see what sticks.
- Adding defensive `if x == nil` guards without knowing why `x` is nil.
- Declaring it fixed because the symptom disappeared once.
- Three failed fixes in a row without a new hypothesis — stop, return to
  Phase 1, and question the architecture or your reproduction instead.
