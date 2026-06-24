---
name: code-formatting
description: >-
  Keep code consistently formatted and lint-clean before completing work. Use
  when finishing an edit, fixing linter/formatter failures, or setting up
  format/lint commands for Go, TypeScript/JavaScript, and Markdown.
---

# Code Formatting & Linting

Formatting is not opinion — run the tool and move on. Never hand-format what a
formatter owns. Always finish a change with formatters and linters clean.

## When to use

- Right before declaring an edit complete.
- A CI or local lint/format check failed.
- Touching a file whose formatting drifted.

## Principles

- **Let tools decide style.** Don't argue spacing/quotes/imports — run the
  formatter.
- **Format only what you touched** unless a repo-wide reformat is explicitly
  requested; avoid unrelated churn in diffs.
- **Lint failures are signal.** A `depguard`/import-rule failure is a design
  violation (wrong dependency), not a nit — fix the import, don't suppress it.
- **Don't blanket-disable rules.** Narrow, justified `//nolint`/`eslint-disable`
  with a reason only when truly warranted.

## Go

```bash
gofmt -w .            # or: go fmt ./...
goimports -w .        # group/order imports
go vet ./...
golangci-lint run     # repo lint config (depguard, etc.)
```

In this workspace, `make lint` runs the configured linters; treat its failures
as blocking. Forbidden-import errors mean "use the RonyKIT equivalent".

## TypeScript / JavaScript (frontend)

```bash
pnpm format           # Prettier (or biome) write
pnpm lint --fix       # ESLint / Biome autofix
pnpm typecheck        # tsc --noEmit
```

Prefer the formatter the repo already configures (Prettier or Biome) — do not
introduce a second one.

## Markdown

```bash
make format-md        # repo Markdown formatter (preserves YAML frontmatter)
```

## Checklist before "done"

- Formatter run on changed files; diff contains no stray reformatting.
- Linter and type checker pass (or only pre-existing, unrelated failures remain).
- No new broad lint suppressions.
