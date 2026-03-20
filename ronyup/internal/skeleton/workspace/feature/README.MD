# Feature Modules

Feature modules are mid-level services that implement business use-cases.
They should expose contracts through `api/` and keep orchestration logic in
`internal/app`.

## Layering Guidelines

- `api/`: transport and contract boundary (DTOs, route selectors, request validation).
- `internal/app/`: business orchestration and use-cases.
- `internal/repo/port.go`: repository abstractions used by app layer.
- `internal/repo/v0/`: concrete adapters (db/cache integrations). Default DB stack is Postgres with `sqlc`.
- `internal/settings/`: runtime settings model and config loading.

Keep API handlers thin. A handler should mostly:
1) validate/normalize inputs,
2) call app use-case methods,
3) map results/errors to response types.

## How to Create a New Workspace

Workspace is the root of your services and should be created once.

- Use `ronyup` CLI to create a new workspace.
- Example: `ronyup setup workspace --repoDir ./your-repo --repoModule github.com/you/your-repo`

## How to Create a New Feature

- Use `ronyup` CLI to create a new feature.
- Current templates: `service` (default; `job` and `gateway` are reserved for future scaffolds).
- Example: `ronyup setup feature --featureDir billing --featureName billing --template service`

## Module Workflow

- After changing contracts in a feature, run `make gen-stub` inside that feature module.
- In other services, import and use those generated stubs for inter-service calls.
- `make tidy` to align dependencies.
- `make lint` to apply formatting and static checks.
- `make test` before opening a PR.

