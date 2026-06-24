---
name: writing-plans
description: >-
  Write a concrete implementation plan before touching code on a multi-step
  task. Use when you have a spec or requirements and the work spans several
  files or steps — decompose into small, independently testable tasks first.
---

# Writing Plans

For any multi-step change, write the plan before the code. A good plan lets
someone with no context implement the work correctly, one small task at a time.

## When to use

- You have a spec/requirements for a feature that touches multiple files or
  steps.
- Before starting non-trivial implementation (skip for one-line fixes).

Save plans to `docs/plans/YYYY-MM-DD-<feature-name>.md` (or the repo's
convention) so they're reviewable and durable.

## 1. Map the files first

Before listing tasks, decide which files are created or modified and the single
responsibility of each. Decomposition decisions get locked in here.

- One clear responsibility per file; prefer small, focused files.
- Files that change together live together — split by responsibility, not layer.
- In an existing codebase, follow the established patterns; don't unilaterally
  restructure.

## 2. Right-size the tasks

A task is the smallest unit that carries its own test cycle and is worth a fresh
reviewer's approval. Fold setup/config/scaffolding into the task whose
deliverable needs them. Each task ends with an independently testable result.

## 3. Break tasks into bite-sized steps

Each step is one action (a couple of minutes), following a test-first loop:

- Write the failing test
- Run it; confirm it fails for the right reason
- Write the minimal code to pass
- Run it; confirm it passes
- Commit

## Plan template

```markdown
# <Feature> Implementation Plan

**Goal:** <one sentence — what this builds>
**Approach:** <2-3 sentences on the design>
**Tech / constraints:** <key libraries, version floors, project-wide rules>

## Task N: <component>

**Files:**
- Create: `exact/path/to/file`
- Modify: `exact/path/to/existing` (lines/area)
- Test: `exact/path/to/test`

**Interfaces:**
- Consumes: <signatures this task relies on from earlier tasks>
- Produces: <function names / types later tasks will use>

- [ ] Step 1 — write the failing test (`<run command>`, expect FAIL)
- [ ] Step 2 — minimal implementation
- [ ] Step 3 — run the test, expect PASS
- [ ] Step 4 — commit
```

## Principles

- **DRY, YAGNI, TDD, frequent commits.**
- State exact paths, commands, and expected outputs — assume zero context.
- If the spec spans independent subsystems, split into one plan per subsystem;
  each plan should produce working, testable software on its own.
- Get the plan reviewed/approved before implementing.

---

_Adapted from the `writing-plans` skill in
[obra/superpowers](https://github.com/obra/superpowers) (MIT)._
