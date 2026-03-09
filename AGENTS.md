# AGENTS.md
## Overview
This file defines practical instructions for coding agents working in this repository.
- Scope: entire repository rooted at this directory.
- Default approach: prefer minimal, targeted changes over broad refactors.
## Project Context
- Project: `ronykit`
- Language: Go (multi-module workspace via `go.work`)
- Main modules: `rony`, `kit`, `flow`, `stub`, `ronyup`, `testenv`
- Sub-module directories: `std/` (clusters and gateways), `x/` (extended utilities)
- Examples: `example/`
## Key Rules
- Use fast search tools (`rg`, `rg --files`) when available.
- Avoid destructive git operations unless explicitly requested.
- Read relevant package/module README files before changing behavior.
- Keep edits scoped to the user request.
- Run focused checks first, then broader checks when needed.
## Tooling
- Install required tools: `make setup`
- Lint modules (excludes `example/`): `make lint`
- Vet modules (excludes `example/`): `make vet`
- Tidy modules (excludes `example/`): `make tidy`
- Run tests (excludes `example/` and `ronyup/`): `make test`
- To test `ronyup` directly: `cd ronyup && go test ./...`
## Code Standards
- Preserve existing architecture and naming conventions.
- Add comments only when logic is non-obvious.
- Do not introduce unrelated formatting churn.
- Update docs when behavior or developer workflow changes.
## Testing
- For logic changes, run targeted tests for affected modules.
- For cross-cutting changes, run `make test` when feasible.
- Report any unrun checks and why they were skipped.
## Git Hygiene
- Do not revert unrelated local changes owned by the user.
- Keep commits atomic and explain intent clearly.
