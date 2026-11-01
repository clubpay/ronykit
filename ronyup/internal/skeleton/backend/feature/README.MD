# Feature modules

Feature modules are mid-level services that implement business use cases. They expose contracts through `api/` and keep orchestration logic in `internal/app`.

## Layering guidelines

- `api/`: Transport and contract boundary for DTOs, route selectors, and request validation.
- `internal/app/`: Business orchestration and use cases.
- `internal/repo/port.go`: Repository abstractions used by the app layer.
- `internal/repo/v0/`: Concrete adapters for database and cache integrations. The default database stack is Postgres with `sqlc`.
- `internal/settings/`: Runtime settings model and configuration loading.

Keep API handlers thin. A handler performs these operations:

1. Validate and normalize inputs.
2. Call app use-case methods.
3. Map results and errors to response types.

## Create a workspace

Create the workspace root one time for your services.

- Use the `ronyup` CLI to create a workspace.
- Example: `ronyup setup workspace --repoDir ./your-repo --repoModule github.com/you/your-repo`
- Add `--kind fullstack` for separate `backend/` and `frontend/` directories. The Go workspace is in `backend/`. `devops/`, `docs/`, and AI configuration stay at the root. Run `setup feature` from `backend/` in this layout.

## Create a feature

- Use the `ronyup` CLI to create a feature.
- The current template is `service` only.
- The default layout is `feature/<name>/`, such as `feature/billing/`.
- Example: `ronyup setup feature --featureDir billing --featureName billing --template service`
- Use `--featurePrefix` to change the parent directory. Use `--groupByTemplate` or `-g` for `feature/<template>/<name>/`.

## Module workflow

- Run `make gen-stub` inside a feature module after contract changes.
- Import and use generated stubs for inter-service calls from other services.
- Run `make tidy` to align dependencies.
- Run `make lint` to apply formatting and static checks.
- Run `make test` before you open a pull request.
