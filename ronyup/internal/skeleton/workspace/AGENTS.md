# AGENTS.md

This file defines practical guidance for AI coding assistants in this workspace.

## Repo Purpose

- Go workspace for RonyKIT services.
- Main folders in generated workspaces:
  - `cmd/service`: service process wiring and bootstrap.
  - `feature/`: business modules.
  - `pkg/`: shared libraries (cross-module, business-agnostic).

## Core Rules

- Keep changes scoped to the requested task.
- Do not edit unrelated files.
- Prefer targeted changes over broad refactors.
- Do not run destructive git operations.
- Run focused checks first, then broader checks.

## Architecture Conventions

- Keep transport concerns in `api/` and business logic in `internal/app/`.
- Keep handlers thin; delegate business behavior to app use-cases.
- Define ports in `internal/repo/port.go`.
- Keep adapter implementations in `internal/repo/v0/`.
- Keep module config in `internal/settings/`.
- Shared utilities belong in `pkg/` and should not depend on feature business logic.

## Service Design Pattern

For a new feature module:
- Add/adjust contracts in `api/service.go`.
- Implement use-cases in `internal/app/app.go` (and split files as it grows).
- Add persistence interfaces in `internal/repo/port.go`.
- Implement datastore integration in `internal/repo/v0/adapter.go`.
- Update settings and config templates when new dependencies are introduced.

## Suggested Workflow

1. Scaffold feature: `ronyup setup feature --featureDir <dir> --featureName <name> --template service`
2. Implement contracts and app use-cases.
3. Add tests for app and API behavior.
4. Run:
   - `make tidy`
   - `make lint`
   - `make test`

## Quality Expectations

- Idempotent behavior for retryable operations.
- Validate input at API boundary.
- Keep domain naming consistent across API, app, repo, and settings.
