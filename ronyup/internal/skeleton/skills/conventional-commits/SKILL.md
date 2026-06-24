---
name: conventional-commits
description: >-
  Write clear, atomic commits using Conventional Commits. Use when committing
  changes, writing or rewriting a commit message, or splitting work into
  reviewable commits.
---

# Conventional Commits

Commits are a changelog and a debugging tool. Keep them atomic and describe the
*why*, using the Conventional Commits format.

## When to use

- Creating any commit or writing a commit message.
- Splitting a branch of work into reviewable, atomic commits.

## Format

```
<type>(<optional scope>): <subject>

<optional body — the why, not the what>

<optional footer — BREAKING CHANGE:, refs #123>
```

- Subject: imperative mood ("add", not "added"), ≤ ~72 chars, no trailing period.
- Body: explain motivation and context; wrap at ~72 columns.
- Breaking changes: `!` after type/scope (`feat!:`) or a `BREAKING CHANGE:` footer.

## Types

| Type | Use |
|------|-----|
| `feat` | New user-facing feature |
| `fix` | Bug fix |
| `refactor` | Code change with no behavior change |
| `perf` | Performance improvement |
| `docs` | Documentation only |
| `test` | Adding or fixing tests |
| `build` | Build system or dependencies |
| `ci` | CI configuration |
| `chore` | Maintenance, tooling, housekeeping |
| `style` | Formatting only (no logic change) |

## Atomic commits

- One logical change per commit; it should build and pass tests on its own.
- Separate refactors from behavior changes into different commits.
- Don't bundle unrelated fixes; split them.

## Examples

```
feat(auth): add refresh-token rotation

Rotate refresh tokens on every use so a leaked token is valid for at
most one request. Old tokens are revoked atomically.

refs #482
```

```
fix(parser): handle empty input without panicking
```

## Before committing

- Review the staged diff; stage only what belongs in this commit.
- Never commit secrets, credentials, or large generated artifacts.
- Don't commit on a protected branch unless explicitly intended.
