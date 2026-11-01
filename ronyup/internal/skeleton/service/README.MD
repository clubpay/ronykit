# Service feature

This module contains a small teaching CRUD slice. Replace its item domain, handlers, app methods, repository port, and in-memory adapter with the behavior in the approved SRS and SDD; keep the established layer boundaries.

## Development commands

- `make test`: run the module tests.
- `make gen-stub`: regenerate client stubs after changing API contracts.
- `make sqlc`: regenerate sqlc code after changing queries or migrations.
- `make verify`: run the feature quality checks from the Go workspace root.

Keep handlers thin, business logic in `internal/app`, and persistence behind `internal/repo/port.go`. Repository port methods need integration tests in `internal/repo/integration_test/`; exported app methods need unit tests. Use the RonyKIT packages selected by the workspace guidance rather than introducing parallel helper libraries.
