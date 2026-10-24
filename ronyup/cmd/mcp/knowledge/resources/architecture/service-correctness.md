Correctness rules for feature code. Each one prevents a bug that compiles, passes the happy-path tests, and only shows up in production.

## Not-found is not a failure

The repo layer turns `sql.ErrNoRows` into the domain's not-found error and wraps every other error with a code that says what happened.
Never give a non-not-found error the `errs.NotFound` code, and never return a default value in place of a real error.

```go
// internal/domain/errors.go
var (
	ErrAccountNotFound = errs.GenWrap(errs.NotFound, "ACCOUNT_NOT_FOUND")
	ErrAccountStore    = errs.GenWrap(errs.Internal, "ACCOUNT_STORE_FAILED")
)

// internal/repo/v0/account.go
row, err := r.q.GetAccount(ctx, id)
if errors.Is(err, sql.ErrNoRows) {
	return nil, domain.ErrAccountNotFound(err)
}
if err != nil {
	return nil, domain.ErrAccountStore(err)
}
```

- Compare errors with `errors.Is` / `errors.As`, or branch on `errs.Code(err)`. Never `==` and never a match on the message text.
- `errs.B().Cause(err)` keeps the cause's code. A handler that sets `.Code(errs.NotFound)` on any error it gets reports a database outage as
  a 404.
- Test it. Next to the not-found integration test, add an `App` unit test where the repo fake returns a plain error and assert
  `errs.Code(err) != errs.NotFound`.

## Identity comes from the caller, not the input

The tenant, merchant, or user an operation acts for comes from the verified token that your auth middleware stored on the request
(`ctx.Get(...)` or values on `ctx.Context()`). Never take it from a field in the input DTO.

- After loading a resource, `internal/app` checks that it belongs to that caller. A path or body ID the caller does not own is
  `errs.NotFound`, so it does not reveal that the resource exists.
- A request with no identity is `errs.Unauthenticated`, not a query without a filter.
- Every repo query on owned data filters on the owner column.

## Authorization covers every handler

`rony.WithMiddleware` applies only to handlers registered **after** it in `Desc()`. A middleware added below a `rony.WithUnary` does not
protect that handler.

`rony.SetupOptionGroup` restores the base path when it ends but **not** the middleware list. Middleware added inside a group also covers
every handler registered after the group.

Put auth middleware first in `Desc()`, and list any public handler explicitly with a comment saying why it is public.

## Context goes all the way down

Handlers pass `ctx.Context()` into `internal/app`, and app methods pass it into repos, stubs, and `flow`. Never start a new
`context.Background()` inside a request path, because it drops cancellation, deadlines, and the trace.

## Concurrent writes

- Never hold a mutex across I/O (a DB call, a stub call, Redis).
- Write `INSERT ... ON CONFLICT` sqlc queries instead of a select followed by an insert. Two requests can both pass the select.
- Serialize expensive side effects per key (a Redis lock or a `flow` workflow ID per resource), not with one global lock.
- See `knowledge://ronyup/characteristics/idempotent` for retry safety.

## Logs and error metadata

- Log with `logkit` and include the IDs needed to trace a request (tenant, resource ID). Never log a secret, a token, card data, or a raw
  request body.
- Error metadata (`errs.WrapCode(..., metaPairs...)`, `GenWrap` extra args) is serialized when errors travel between services. Never put a
  secret in it.
