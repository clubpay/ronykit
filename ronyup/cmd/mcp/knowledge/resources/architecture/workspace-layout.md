`ronyup setup workspace` supports three layouts via `--kind` (default `backend`):

- **`backend`** — a Go-only workspace at the repository root.
- **`fullstack`** — a `backend/` + `frontend/` split. The Go workspace is moved into `backend/`, while AI assistant config (`.ai/`, `.agents/`, `.cursor/`, `AGENTS.md`), `devops/`, and `docs/` stay at the repository root and are shared by both sides.
- **`frontend`** — a frontend-only workspace: a `frontend/` application plus shared AI assistant config (`.ai/`, `.agents/`, `.cursor/`, `AGENTS.md`) and `docs/` at the repository root. There is **no** Go workspace, `devops/`, `Makefile`, or backend verify gate; only the frontend verify stop hook is installed.

## Backend layout (root of the Go workspace)

In `backend` kind this is the repository root; in `fullstack` kind it is `backend/`:

- `go.work` — lists every Go module in the workspace.
- `cmd/service/` — the executable entrypoint module (cobra-based). `main.go` builds a `rony.Server`, wires observability exporters (`tracekit`, `meterkit`, `logkit`), and starts every registered service via `di.AllServices()` (or a filtered subset via `di.GetService("service", name)`).
- `cmd/service/features.go` — a `package main` file whose blank imports trigger each feature module's `init()` (which calls `di.RegisterService`). `ronyup setup feature` rewrites this file when adding a feature.
- `cmd/service/middleware.go` — registers global middlewares (panic recovery, base headers, tracing/logging) via `di.RegisterMiddleware` in `init()`.
- `feature/` — business modules (default parent directory; override with `--featurePrefix`). By default, a service named `auth` lives at `feature/auth/`. With `--groupByTemplate`, it lives at `feature/service/auth/`; job and gateway templates would use `feature/job/<name>/` or `feature/gateway/<name>/`.
- `pkg/` — shared internal libraries (`pkg/i18n` is created by `setup workspace`; add others such as `bkit`, `log`, `datasource`, `msg` as needed). Keep `pkg/*` free of feature-specific business logic.
- `.golangci.yml` — package-selection enforcement (depguard).
- `Makefile` — workspace tasks (`make run`, `make test`, `make lint`, …).

Each module under `feature/<name>/` (or `feature/<template>/<name>/` when grouped) and `pkg/<name>/` is an independent Go module with its own `go.mod`. The workspace `go.work` lists every module; `ronyup setup feature` runs `go work use .` for the new feature.

## Shared root layout (always at the repository root)

- `devops/` — `devops/devbox/` installs optional platform services (Postgres, Redis, Temporal, RustFS/S3, …) into a Kubernetes cluster via Helm. Default `cluster.mode: existing` uses your kubeconfig; set `vagrant` for a local microk8s VM.
- `docs/` — design documents (`docs/design/<feature>-srs.md`, `…-sdd.md`) and guides.
- `.ai/mcp/mcp.json`, `.cursor/mcp.json` — preconfigured client config so IDEs pick up the `ronyup mcp` server.
- `.agents/skills/ronykit-framework/` — the agent skill.
- `AGENTS.md` — agent guidance.

## Module paths and command location

- In `backend` kind, modules are `<repoModule>/cmd/service`, `<repoModule>/feature/<name>`, etc.
- In `fullstack` kind, the Go workspace is nested, so modules are `<repoModule>/backend/cmd/service`, `<repoModule>/backend/feature/<name>`, etc. Run `ronyup setup feature` (and `go`/`make` commands) from the `backend/` directory. `docs/design` still lives at the repository root, and the `scaffold_feature` design gate resolves it from the parent of `backend/` automatically.

## Frontend (`fullstack` and `frontend` kinds)

- `frontend/` — holds the web/mobile application(s) at the repository root. It is framework-agnostic by default (a placeholder `README.MD`); initialize it with the stack of your choice (React/Vite, Next.js, SvelteKit, …). In `fullstack` workspaces it calls the backend via its OpenAPI spec at `/docs`; in `frontend`-only workspaces it talks to external services over their HTTP/OpenAPI APIs.
- **One app vs. many — always clarify first.** Do not assume a single frontend app. Before creating or editing anything under `frontend/`, ask the user whether there is one app or multiple.
  - Single app: code may live directly under `frontend/`.
  - Multiple apps: give each app its own directory, `frontend/<app-name>/` (e.g. `frontend/admin/`, `frontend/web/`). Confirm which app a change targets, and the app name/stack when initializing a new one, before proceeding.
