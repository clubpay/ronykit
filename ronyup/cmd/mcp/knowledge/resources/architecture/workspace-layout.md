A ronyup-scaffolded workspace has this top-level layout:

- `cmd/service/` — the executable entrypoint module (cobra-based). `main.go` builds a `rony.Server`, wires observability exporters (`tracekit`, `meterkit`, `logkit`), and starts every registered service via `di.AllServices()` (or a filtered subset via `di.GetService("service", name)`).
- `cmd/service/features.go` — a `package main` file whose blank imports trigger each feature module's `init()` (which calls `di.RegisterService`). `ronyup setup feature` rewrites this file when adding a feature.
- `cmd/service/middleware.go` — registers global middlewares (panic recovery, base headers, tracing/logging) via `di.RegisterMiddleware` in `init()`.
- `feature/` — business modules (default parent directory; override with `--featurePrefix`). By default, a service named `auth` lives at `feature/auth/`. With `--groupByTemplate`, it lives at `feature/service/auth/`; job and gateway templates would use `feature/job/<name>/` or `feature/gateway/<name>/`.
- `pkg/` — shared internal libraries (`pkg/i18n` is created by `setup workspace`; add others such as `bkit`, `log`, `datasource`, `msg` as needed). Keep `pkg/*` free of feature-specific business logic.
- `devops/` — Docker Compose and deployment helpers seeded by the scaffold.
- `.ai/mcp/mcp.json` — preconfigured client config so IDEs pick up the `ronyup mcp` server.

Each module under `feature/<name>/` (or `feature/<template>/<name>/` when grouped) and `pkg/<name>/` is an independent Go module with its own `go.mod`. The workspace `go.work` lists every module; `ronyup setup feature` runs `go work use .` for the new feature.
