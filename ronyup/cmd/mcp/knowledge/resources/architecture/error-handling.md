Use the `rony/errs` package for all error definitions and construction.

Primary patterns:

1. `errs.GenWrap(code, "ERROR_CODE")` for wrappable sentinel errors
   that accept additional context via `err("detail text")`. `code` is a
   value of type `errs.ErrCode`.
2. `errs.B().Code(code).Msg("ERROR_CODE").Err()` for static sentinel
   errors.

Error codes are `SCREAMING_SNAKE_CASE` strings (for example
`REFRESH_TOKEN_EXPIRED`, `ACCOUNT_NOT_FOUND`, `SERVICE_FAILED`).

In API handlers, wrap errors from the app layer with:

- `errs.B().Cause(err).Msg("OPERATION_FAILED").Err()`.

Common `errs.ErrCode` values (with the HTTP status the framework maps them to):

- `errs.InvalidArgument` (400)
- `errs.Unauthenticated` (401)
- `errs.PermissionDenied` (403)
- `errs.NotFound` (404)
- `errs.AlreadyExists` (409)
- `errs.Aborted` (409)
- `errs.FailedPrecondition` (400)
- `errs.ResourceExhausted` (429)
- `errs.Internal` (500)
- `errs.DeadlineExceeded` (504)

Place all domain-specific errors in `internal/domain/errors.go` as package-level
variables.
