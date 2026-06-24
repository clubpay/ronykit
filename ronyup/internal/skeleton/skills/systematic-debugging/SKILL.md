---
name: systematic-debugging
description: >-
  Find the root cause before changing code. Use when encountering any bug, test
  failure, crash, flaky test, performance regression, or unexpected behavior,
  before proposing or applying a fix.
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
  `-race` / repeated runs.

## Anti-patterns

- Trying random changes to see what sticks.
- Adding defensive `if x == nil` guards without knowing why `x` is nil.
- Declaring it fixed because the symptom disappeared once.
