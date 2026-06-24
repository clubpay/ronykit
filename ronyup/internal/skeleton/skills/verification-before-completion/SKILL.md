---
name: verification-before-completion
description: >-
  Require fresh verification evidence before claiming work is done, fixed, or
  passing. Use before reporting completion, expressing success, committing, or
  opening a PR — run the command, read the output, then make the claim.
---

# Verification Before Completion

Claiming work is complete without verifying it is guesswork, not efficiency.
Evidence before claims, always.

## The rule

> No completion claim without fresh verification evidence.

If you have not run the verifying command for the change in front of you, you
cannot say it passes, builds, or is fixed.

## The gate (run before any "done")

1. **Identify** the command that proves the claim.
2. **Run** the full command fresh — not a partial or cached run.
3. **Read** the full output: exit code, failure count, warnings.
4. **Verify** the output actually confirms the claim.
5. **Then** state the result, with the evidence.

## What each claim actually requires

| Claim | Sufficient evidence | Not enough |
|-------|---------------------|-----------|
| Tests pass | Test output: 0 failures | "should pass", a previous run |
| Linter clean | Linter output: 0 errors | a partial check |
| Build succeeds | Build exits 0 | linter passing, "logs look fine" |
| Bug fixed | The original symptom no longer reproduces | code changed, assumed fixed |
| Regression test works | Red→green verified (fails without the fix) | test passes once |
| Requirements met | Line-by-line check against the spec | "tests pass, so we're done" |
| Delegated work done | Diff/output inspected directly | the agent reported success |

## Red flags — stop and verify

- Words like "should", "probably", "seems to", or celebrating before running
  anything ("Done!", "Perfect!").
- About to commit, push, or open a PR without a fresh run.
- Trusting a subagent's "success" without checking the diff.
- "Just this once" / "I'm confident" / "the linter passed so it compiles".

## Regression tests: prove red→green

A regression test that only passes proves nothing. Confirm it fails without the
fix:

```
write test → run (fails) → apply fix → run (passes)
```

## Bottom line

Run the command. Read the output. Then make the claim. No shortcuts.

---

_Adapted from the `verification-before-completion` skill in
[obra/superpowers](https://github.com/obra/superpowers) (MIT)._
