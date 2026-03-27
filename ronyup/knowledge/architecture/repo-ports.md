Define persistence interfaces in `internal/repo/port.go` using domain types
(never API types).

- Create separate interfaces per domain concept
  (`AccountRepository`, `JWTRepository`, `PolicyRepository`).
- Larger services may compose interfaces
  (`LedgerRepository` embeds `AccountRepository + TransferRepository`).

In `internal/repo/v0/adapter.go`, declare:

- `var Init = fx.Options(fx.Provide(fx.Private, db.New, fx.Annotate(NewXRepo, fx.As(new(repo.XRepository))), ...))`
  to bind concrete implementations to interfaces.

Use `fx.Private` to prevent leaking concrete types outside the repo module.

Implement each concrete repository in a separate file (`v0/account.go`,
`v0/jwt.go`, etc.). The `db.New` function creates the sqlc `Queries` struct
from the database pool.
