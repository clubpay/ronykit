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

The scaffolded `backend/.golangci.yml` (golangci-lint v2) owns both linting
(`depguard`) and formatting (`gofmt`, `gofumpt`, `goimports`, `gci`). Run it
rather than the individual tools, so import grouping matches the config:

```bash
make lint             # every module: golangci-lint run --fix ./...
make vet              # every module: go vet ./...
cd feature/<name> && golangci-lint fmt ./...   # format one module only
```

Treat `make lint` failures as blocking. A `depguard` error means "use the
RonyKIT equivalent" (see `knowledge://ronyup/architecture/package-selection`),
not "add a `//nolint`". Generated code (sqlc `data/db`, stubs) is regenerated
with `make sqlc` / `make gen-stub`, never hand-formatted.

## TypeScript / JavaScript (frontend)

Use the scripts the app's `package.json` defines (names vary; check first):

```bash
pnpm format           # Prettier (or biome) write
pnpm lint --fix       # ESLint / Biome autofix
pnpm typecheck        # tsc --noEmit
bash frontend/verify.sh   # the full gate: typecheck, lint, build, test, stories
```

Prefer the formatter the repo already configures (Prettier or Biome) — do not
introduce a second one.

## Markdown

Use the repo's Markdown formatter if one is configured (a `make format-md`
target, Prettier, or `markdownlint`); otherwise leave Markdown formatting as
is. Never let a formatter rewrite the YAML frontmatter of design documents or
`SKILL.md` files — the `scaffold_feature` design gate and skill discovery parse
it.

## Checklist before "done"

- Formatter run on changed files; diff contains no stray reformatting.
- Linter and type checker pass (or only pre-existing, unrelated failures remain).
- No new broad lint suppressions.
