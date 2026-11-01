# Frontend

This directory contains the frontend application for **{{ .ApplicationName }}**.

`ronyup setup workspace --kind fullstack` created this directory. The project has a `backend/` Go workspace and this `frontend/` application. Shared resources stay at the repository root:

```
<repo-root>/
├── backend/    # RonyKIT Go workspace (go.work, cmd/all-in-one, feature/, pkg/)
├── frontend/   # this directory — your web/mobile app
├── devops/     # devbox: Kubernetes platform services (shared)
├── docs/       # design docs (SRS/SDD) and guides (shared)
├── AGENTS.md   # agent guidance (shared)
└── .ai/ .agents/ .cursor/   # AI assistant config (shared)
```

## One app or many?

Select the topology before initialization:

- **Single app:** Initialize the app directly in `frontend/`.
- **Multiple apps:** Give each app a subdirectory, `frontend/<app-name>/`. Examples are `frontend/web/` and `frontend/admin/`. Initialize each app in its subdirectory.

AI assistants must ask which case applies. They must also ask for the target app's name and stack before creating or editing frontend code. Do not assume a single app.

## Design-first bootstrap

Complete these steps before `create-next-app`, `create vite`, or any UI code:

1. Ask design questions about audience, mood, brand, light or dark mode, and v1 screens.
2. Write `docs/design/<app>-frontend-design.md`. Use MCP `design-frontend` and skills `frontend-design`, `design-tokens`, and `typography`. Use slug `web` for one app at `frontend/`. Use the directory name for `frontend/<app>/`.
3. Get user approval. The frontmatter must contain `status: approved`.

After `package.json` exists, `verify.sh` checks the approved design document.

## Getting started

This directory is framework-agnostic. Initialize each app with your selected stack.

Run these commands in `frontend/` for one app. For multiple apps, run them in each `frontend/<app-name>/` directory:

```bash
# React + Vite
npm create vite@latest .

# Next.js
npx create-next-app@latest .

# SvelteKit
npm create svelte@latest .
```

## Verification gate

Run the verification gate before reporting frontend work as complete. `verify.sh` defines the frontend quality checks.

The script detects one app in this directory or multiple apps under `frontend/<app>/`. It runs the scripts defined by each app: `typecheck`, `lint`, `build`, `test`, and `build-storybook`. It also runs static UI polish and UX checks. Every UI component must have a co-located `*.stories.tsx` file.

Until an app is initialized, the script performs no checks.

```bash
make verify                 # or: bash verify.sh
SKIP_STORIES_CHECK=1 make verify-no-stories   # skip stories-coverage
```

Define these npm scripts in each app's `package.json`: `typecheck`, `lint`, `build`, and `test`. After Storybook setup, also define `build-storybook`.

Cursor and Claude Code `stop` hooks run the same gate. These hooks are `.cursor/hooks/frontend-verify.sh` and `.claude/hooks/frontend-verify.sh`.

When an agent changes `frontend/`, the hook runs `verify.sh` again. The agent must fix failures before completion.

## Backend communication

- Start the backend with `cd ../backend && make run`. The service exposes an OpenAPI/Swagger specification at `/docs`.
- Generate a typed client from the backend stubs. You can also call the documented HTTP endpoints directly.
- Install shared infrastructure, including Postgres and Redis, with `../devops/devbox/`. Run `make devbox-up` from `../backend` or `make -C ../devops/devbox up`.
