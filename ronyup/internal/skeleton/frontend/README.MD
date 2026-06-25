# Frontend

This directory holds the frontend application for **{{ .ApplicationName }}**.

It was created by `ronyup setup workspace --kind fullstack`, which splits the
project into a `backend/` Go workspace and this `frontend/` application while
keeping shared concerns at the repository root:

```
<repo-root>/
├── backend/    # RonyKIT Go workspace (go.work, cmd/service, feature/, pkg/)
├── frontend/   # this directory — your web/mobile app
├── devops/     # docker-compose and deployment helpers (shared)
├── docs/       # design docs (SRS/SDD) and guides (shared)
├── AGENTS.md   # agent guidance (shared)
└── .ai/ .agents/ .cursor/   # AI assistant config (shared)
```

## One app or many?

Decide the topology before initializing anything here:

- **Single app** — initialize directly in this `frontend/` directory.
- **Multiple apps** — give each app its own subdirectory, `frontend/<app-name>/`
  (for example `frontend/web/`, `frontend/admin/`), and initialize each one
  there.

AI assistants must **ask** which case applies (and the target app's name/stack)
before creating or editing frontend code — they should not assume a single app.

## Design-first bootstrap (enforced)

Before running `create-next-app`, `create vite`, or writing any UI:

1. Ask aesthetic/design questions (audience, mood, brand, light/dark, v1 screens).
2. Write `docs/design/<app>-frontend-design.md` using MCP `design-frontend` and skills `frontend-design`, `design-tokens`, `typography`.
   Use slug `web` for a single app at `frontend/`, or the directory name for `frontend/<app>/`.
3. Get user approval (`status: approved` in frontmatter).

`verify.sh` checks the approved design doc once `package.json` exists.

## Getting started

This folder is intentionally framework-agnostic. Initialize your app(s) with the
stack of your choice. Run these in `frontend/` for a single app, or in
`frontend/<app-name>/` for each app when you have multiple:

```bash
# React + Vite
npm create vite@latest .

# Next.js
npx create-next-app@latest .

# SvelteKit
npm create svelte@latest .
```

## Verify gate (run before calling frontend work "done")

`verify.sh` is the single source of truth for frontend quality. It detects each
app (this directory for a single app, or each `frontend/<app>/` for multiple),
runs whatever scripts the app defines (`typecheck`, `lint`, `build`, `test`,
`build-storybook`), and checks that every UI component has a co-located
`*.stories.tsx`. It is a no-op until an app is initialized.

```bash
make verify                 # or: bash verify.sh
SKIP_STORIES_CHECK=1 make verify-no-stories   # skip stories-coverage
```

So the gate works, expose these npm scripts in each app's `package.json`:
`typecheck`, `lint`, `build`, `test`, and (once Storybook is set up)
`build-storybook`.

This gate is also wired into a Cursor `stop` hook
(`.cursor/hooks/frontend-verify.sh`): when an agent changes anything under
`frontend/`, the hook re-runs `verify.sh` and loops the agent back to fix
failures automatically — no manual "please fix this" round-trips.

## Talking to the backend

- The backend service exposes an OpenAPI/Swagger spec at `/docs` when running
  (`cd ../backend && make run`).
- Generate a typed client from the backend stubs, or call the documented HTTP
  endpoints directly.
- Shared infrastructure (Postgres, Redis, …) is defined in `../devops/docker-compose.yml`.
