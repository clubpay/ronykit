---
name: check_rkit_usage
---

Report Go code that re-implements an `x/rkit` helper, and optionally rewrite it to call rkit. This is the mechanical half of
`architecture/package-selection`: prose tells you to reach for rkit, this check catches the cases where code didn't.

## When to use

- After writing or editing Go code in a feature module, before claiming the work is done.
- When `make verify` fails with `ronyup lint` findings.
- When reviewing a feature for package-selection compliance (`review-architecture`).

## Extended Guidance

Available as the MCP tool `check_rkit_usage` (in-process; equivalent to `ronyup lint`):

| Argument     | CLI flag        | Default     |
|--------------|-----------------|-------------|
| `path`       | (working dir)   | (required)  |
| `patterns`   | positional args | `["./..."]` |
| `fix`        | `--fix`         | `false`     |
| `tests`      | `--tests`       | `false`     |
| `allModules` | `--all-modules` | `false`     |

CLI equivalent:

```bash
ronyup lint [--fix] [--tests] [--all-modules] [packages]
```

Exit status: `0` clean, `1` findings left, `2` packages do not load or type-check.

## Rules

| Rule         | Matches                                                         | Fix                                     |
|--------------|-----------------------------------------------------------------|-----------------------------------------|
| `must`       | `v, err := f(); if err != nil { panic(err) }` (err used nowhere else) | `v := rkit.Must(f())` / `rkit.Assert(f())` |
| `mapcollect` | a slice declaration plus a loop that only appends a map's keys or values | `rkit.MapKeysToArray(m)` / `rkit.MapToArray(m)` |
| `unsafeconv` | `unsafe.String(unsafe.SliceData(b), len(b))`, `unsafe.Slice(unsafe.StringData(s), len(s))`, `*(*string)(unsafe.Pointer(&b))` | `rkit.B2S(b)` / `rkit.S2B(s)` |
| `helper`     | an unexported function whose body re-implements `rkit.Must`, `PtrVal`, `Contains` or `Coalesce` | none — delete it and call rkit |

A directly following `return v` is folded into the fix. Fixes are skipped (the finding stays) when a comment sits inside the
rewritten range, or when `var s []T` would change from nil to an empty slice.

## Behaviour

- Modules whose `go.mod` does not require `github.com/clubpay/ronykit/x/rkit` are skipped unless `allModules` is set; rkit is
  not worth a new dependency for a one-line helper.
- Generated files (`// Code generated ... DO NOT EDIT.`) are skipped.
- Silence a deliberate exception with `//rkitcheck:ignore <reason>` on the reported line or the line above.
- In scaffolded workspaces, `verify.sh` runs `ronyup lint` on every feature module, and the Claude Code `PostToolUse` hook
  (`.claude/hooks/backend-lint.sh`) runs it on each Go file the agent edits.
