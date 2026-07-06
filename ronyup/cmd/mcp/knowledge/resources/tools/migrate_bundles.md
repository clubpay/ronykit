---
name: migrate_bundles
---

Upgrade an **existing** workspace created before executable bundles to the current layout (`pkg/runner`, `bundles.yaml`, thin `cmd/all-in-one/main.go`).

## When to use

- The workspace was scaffolded with an older `ronyup` that inlined bootstrap code in `cmd/service/main.go` or `cmd/all-in-one/main.go`.
- The repo is missing `pkg/runner/`, `bundles.yaml`, or still has legacy middleware/healthz files under the default bundle directory.
- The repo still uses the intermediate `cmd/runner/` layout and needs bootstrap code in `pkg/runner/`.
- The repo still uses the legacy `cmd/service/` bundle name and needs `cmd/all-in-one/`.
- You upgraded `ronyup` and want bundle support without re-scaffolding the repository.

## Command

```bash
ronyup setup migrate bundles [--dry-run]
```

Run from the Go workspace root (directory with `go.work`) **or** from the repository root in a fullstack workspace (`backend/go.work` is resolved automatically).

## Behaviour

- **Idempotent** — safe to run multiple times. If already migrated, refreshes bundle `features.go` files only.
- **Does not touch** feature business code under `feature/*` or other `pkg/*` modules (except `go work use` for `pkg/runner`).
- Creates a backup `cmd/all-in-one/main.go.legacy` when replacing a monolithic main.
- After migration, use `ronyup setup bundle` to add production bundles and `ronyup setup sync --only backend` for Makefile targets.

## Steps performed

1. Copy `pkg/runner/` from the embedded scaffold (shared bootstrap).
2. Rename legacy `cmd/service/` to `cmd/all-in-one/` when present.
3. Rewrite `cmd/all-in-one/main.go` to delegate to `pkg/runner`.
4. Remove legacy middleware/healthz files from the default bundle directory.
5. Remove legacy `internal/runner/` and `cmd/runner/` when upgrading from older layouts.
6. Create `bundles.yaml` when missing (default `all-in-one` bundle with `"*"`).
7. Initialize `pkg/runner` module, `go work use ./pkg/runner`, and regenerate bundle import lists.

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
