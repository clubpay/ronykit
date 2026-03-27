Organize the workspace into four top-level directories:

- `core/` for core business services (`auth`, `user`, `ledger`, `notification`, `storage`).
- `feature/` for feature-specific services (`payment`, `kyc`, `otc`, `portfolio`, etc.).
- `gateway/` for API gateway modules that compose core and feature services for
  different audiences (`admin`, `user`, `agent`).
- `pkg/` for shared internal libraries (`bkit`, `testkit`, `log`, `datasource`,
  `di`, `i18n`, `msg`).

Each service under `core/` and `feature/` is an independent Go module with its
own `go.mod`. The `go.work` file at the workspace root lists all modules.

The `cmd/` directory contains executable entrypoints (for example,
`cmd/all-in-one` for the monolith binary). The all-in-one binary blank-imports
all service modules in a `modules.go` file, which triggers their `init()`
registrations via `di.RegisterService`, and discovers them at runtime via
`di.GetServiceByKind`.
