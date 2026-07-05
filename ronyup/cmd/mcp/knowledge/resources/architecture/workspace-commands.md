---
name: workspace_commands
---

Where to run `ronyup setup` subcommands in scaffolded workspaces.

## Go workspace root

These commands operate on the directory that contains `go.work` (feature modules, bundles, migration). They also accept the **repository root** in fullstack workspaces when `backend/go.work` exists — the CLI resolves `backend/` automatically.

| Command | Run from |
|---------|----------|
| `setup feature` | Go workspace root **or** fullstack repo root |
| `setup bundle` | Go workspace root **or** fullstack repo root |
| `setup migrate bundles` | Go workspace root **or** fullstack repo root |
| `make`, `go work`, `go test` | Go workspace root (`backend/` in fullstack) |

## Repository root

These commands target shared files outside the Go workspace (AI config, devops, docs) or create a new tree.

| Command | Run from |
|---------|----------|
| `setup workspace` | Any directory (creates `--repoDir`) |
| `setup sync` | Repository root **or** `backend/` in fullstack |

## Fullstack summary

```
my-repo/                 ← setup sync, docs/, devops/, AGENTS.md
├── backend/             ← go.work lives here; make/go/feature/bundle/migrate
│   ├── cmd/runner/
│   ├── cmd/service/
│   ├── bundles.yaml
│   └── feature/
└── frontend/
```

When in doubt for backend work: `cd backend` (fullstack) or stay at the repo root (backend-only).
