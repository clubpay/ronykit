---
name: migrate_bundles
---

Upgrade an **existing** workspace created before executable bundles to the current layout (`cmd/runner`, `bundles.yaml`, thin `cmd/service/main.go`).

## When to use

- The workspace was scaffolded with an older `ronyup` that inlined bootstrap code in `cmd/service/main.go`.
- The repo is missing `cmd/runner/`, `bundles.yaml`, or still has `cmd/service/middleware.go` / `healthz.go`.
- You upgraded `ronyup` and want bundle support without re-scaffolding the repository.

## Command

```bash
ronyup setup migrate bundles [--dry-run]
```

Run from the Go workspace root (directory with `go.work`) **or** from the repository root in a fullstack workspace (`backend/go.work` is resolved automatically).

## Behaviour

- **Idempotent** — safe to run multiple times. If already migrated, refreshes bundle `features.go` files only.
- **Does not touch** feature business code under `feature/*` or `pkg/*` (except `go work use` for `cmd/runner`).
- Creates a backup `cmd/service/main.go.legacy` when replacing a monolithic main.
- After migration, use `ronyup setup bundle` to add production bundles and `ronyup setup sync --only backend` for Makefile targets.

## Steps performed

1. Copy `cmd/runner/` from the embedded scaffold (shared bootstrap).
2. Rewrite `cmd/service/main.go` to delegate to `cmd/runner`.
3. Remove legacy `cmd/service/middleware.go` and `healthz.go`.
4. Remove legacy `internal/runner/` when upgrading from an intermediate layout.
5. Create `bundles.yaml` when missing (default `service` bundle with `"*"`).
6. Initialize `cmd/runner` module, `go work use ./cmd/runner`, and regenerate bundle import lists.

## Examples

```bash
# Preview changes
ronyup setup migrate bundles --dry-run

# Apply migration
ronyup setup migrate bundles

# Refresh Makefile bundle targets after migration
ronyup setup sync --only backend
```

## Related

- `knowledge://ronyup/tools/sync_workspace` — refresh AGENTS.md, Makefile, devops (does not perform bundle migration).
- `knowledge://ronyup/tools/setup_bundle` — create additional production bundles after migration.
- `knowledge://ronyup/architecture/workspace-layout` — current workspace layout reference.
- `knowledge://ronyup/architecture/workspace-commands` — where to run setup commands.
