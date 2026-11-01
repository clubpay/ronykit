# Service feature

This module contains a small teaching CRUD slice. Replace the item domain, handlers, app methods, repository port, and in-memory adapter with the approved SRS and SDD behavior. Keep the established layer boundaries.

## Development commands

- `make test`: Run the module tests.
- `make gen-stub`: Regenerate client stubs after API contract changes.
- `make sqlc`: Regenerate sqlc code after query or migration changes.
- `make verify`: Run feature quality checks from the Go workspace root.

Keep handlers thin. Put business logic in `internal/app`. Define persistence interfaces in `internal/repo/port.go`.

Add integration tests for repository port methods in `internal/repo/integration_test/`. Add unit tests for exported app methods.

Use the RonyKIT packages selected by the workspace guidance. Do not introduce parallel helper libraries.
