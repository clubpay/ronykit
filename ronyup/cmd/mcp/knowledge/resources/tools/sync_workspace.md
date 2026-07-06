---
name: sync_workspace
---

Refresh scaffold-managed files in an **existing** workspace from the embedded skeleton shipped with the current `ronyup` binary.

## When to use

- The workspace was created with an older `ronyup` and is missing newer boilerplate (for example `devops/devbox/`, stop hooks, or skill updates).
- You want to refresh `AGENTS.md`, Makefiles, or devops assets without re-scaffolding the repo.

## Command

```bash
ronyup setup sync [--repoDir .] [--only all] [--overwrite] [--skills installed]
```

Run from the repository root, from `backend/` in a fullstack workspace, or from a backend-only Go workspace root.

## Behaviour

- **Does not touch** application code: `cmd/all-in-one/main.go`, `feature/*` modules, `pkg/*` (except scaffold README), or user design docs under `docs/design/`.
- For the bundle + `pkg/runner` layout, run `ronyup setup migrate bundles` once after upgrading `ronyup`.
- **Default**: add missing scaffold files only (`--overwrite` replaces existing scaffold files).
- **Kind**: auto-detected (`backend`, `fullstack`, `frontend`); override with `--kind` if needed.

## Sections (`--only`)

| Section | Path(s) |
|---------|---------|
| `agents` | `AGENTS.md` |
| `ai` | `.ai/`, `.cursor/mcp.json` |
| `hooks` | `.cursor/hooks/`, `hooks.json` |
| `devops` | `devops/devbox/` |
| `docs` | `docs/design/README.MD` |
| `skills` | `.agents/skills/ronykit-framework` + `--skills` selection |
| `backend` | `Makefile`, `verify.sh`, `.golangci.yml`, `bundles.yaml` (when missing) |
| `frontend` | `frontend/Makefile`, `verify.sh`, `README.MD` |
| `all` | all applicable sections (default) |

## Examples

```bash
# Pick up new devbox scaffolding only
ronyup setup sync --only devops

# Refresh agent guidance after upgrading ronyup
ronyup setup sync --only agents --overwrite

# Update installed agent skills
ronyup setup sync --only skills --overwrite --skills installed

# Migrate an older workspace to bundle layout (one-time)
ronyup setup migrate bundles
```
