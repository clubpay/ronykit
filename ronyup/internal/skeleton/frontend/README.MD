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

## Getting started

This folder is intentionally framework-agnostic. Initialize it with the stack
of your choice, for example:

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
