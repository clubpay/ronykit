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

## Talking to the backend

- The backend service exposes an OpenAPI/Swagger spec at `/docs` when running
  (`cd ../backend && make run`).
- Generate a typed client from the backend stubs, or call the documented HTTP
  endpoints directly.
- Shared infrastructure (Postgres, Redis, …) is defined in `../devops/docker-compose.yml`.
