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
| `coalesce`   | runs of `if x == <zero> { x = y }`, and `if a != <zero> { return a }` chains ending the block (fallbacks must be side-effect free) | `x = rkit.Coalesce(x, y, ...)` / `return rkit.Coalesce(a, b, c)` |
| `transform`  | a slice declaration plus a loop over a slice that only appends `f(x)` (or sets `out[i] = f(x)`), or only appends `x` under a condition | `rkit.Map(s, f)` / `rkit.Filter(s, func(x T) bool { ... })` |
| `mapcollect` | a slice declaration plus a loop that only appends a map's keys or values | `rkit.MapKeysToArray(m)` / `rkit.MapToArray(m)` |
| `discard`    | `v, _ := f()` where f returns `(T, error)`; `v, _ := x.(T)` | `rkit.Ok(f())`, or `StrToInt`/`StrToInt64`/`StrToUInt64`/`StrToFloat64`/`ToJSON` for `strconv` parsers and `json.Marshal`; `rkit.TryCast[T](x)` |
| `stdlib`     | `slices.Contains`, `cmp.Or`, `strconv.Itoa`, `strconv.FormatInt/FormatUint(x, 10)`, `fmt.Sprintf("%d", x)`, `slices.Collect(maps.Keys/Values(m))` | `rkit.Contains`, `Coalesce`, `IntToStr`, `Int64ToStr`/`UInt64ToStr`, `IntToStr` & co.; the `Collect` forms are reported only (rkit returns `[]`, not `nil`) |
| `unsafeconv` | `unsafe.String(unsafe.SliceData(b), len(b))`, `unsafe.Slice(unsafe.StringData(s), len(s))`, `*(*string)(unsafe.Pointer(&b))` | `rkit.B2S(b)` / `rkit.S2B(s)` |
| `helper`     | an unexported function whose body re-implements `rkit.Must`, `PtrVal`, `Contains` or `Coalesce` | none — delete it and call rkit |

A directly following `return v` is folded into the fix. Fixes are skipped (the finding stays) when a comment sits inside the
rewritten range, or when `var s []T` would change from nil to an empty slice. Imports a fix leaves unused are removed, and
overlapping fixes are applied over up to three rounds.

## Behaviour

- Modules whose `go.mod` does not require `github.com/clubpay/ronykit/x/rkit` are skipped unless `allModules` is set; rkit is
  not worth a new dependency for a one-line helper.
- Generated files (`// Code generated ... DO NOT EDIT.`) are skipped.
- Silence a deliberate exception with `//rkitcheck:ignore <reason>` on the reported line or the line above.
- In scaffolded workspaces, `verify.sh` runs `ronyup lint` on every feature module, and the Claude Code `PostToolUse` hook
  (`.claude/hooks/backend-lint.sh`) runs it on each Go file the agent edits.
