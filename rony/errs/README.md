# rony/errs: Structured errors and service conventions

`rony/errs` supplies RonyKIT error operations. These operations include the `Error` type, builder, gRPC-style codes, wrapping, HTTP mapping, and serialization.

The package also provides optional conventions for a stable client-facing error contract.

## Wire contract

An `*errs.Error` can represent each error that crosses an API boundary. Example REST body:

```json
{"code": 3, "codeName": "invalid_argument", "item": "TRIGGER_GITHUB_ORG_REQUIRED", "details": null}
```

| Field | Role |
|-------|------|
| `code` | gRPC-style code. The edge derives the HTTP status from this code. Do not create custom codes. |
| `codeName` | String form of the code |
| `item` | Stable machine-readable token in SCREAMING_SNAKE format. Namespace the token by area. Clients can branch on this token. Do not rename a released item. |
| `details` | Client-visible structured context. Use `Reason` or an `errs.ErrDetails` struct. |
| `meta` | Internal diagnostics. External clients do not receive this field. Trusted internal serialization can propagate it. |

External clients do not receive the cause chain. They receive only the boundary error's `item`.

Put all required client data in `item` and `details`. Keep operator diagnostics in the cause chain and `meta`.

## Layer conventions

```go
// Domain: sentinels once, specific failures everywhere.
var ErrInvalidArgument = errs.Sentinel(errs.InvalidArgument, "INVALID_ARGUMENT")

if spec.Org == "" {
	return errs.Specify(ErrInvalidArgument, "TRIGGER_GITHUB_ORG_REQUIRED")
}

// App: annotate, do not rebuild. Wire identity is untouched.
return errs.Annotate(err, "automation.Create")

// API: normalize at the boundary.
return errs.Boundary(err, "AUTOMATION_INTERNAL")
```

- `Specify` keeps the sentinel in the cause chain. `errors.Is` continues to work. The error carries a specific item for clients.
- `Annotate` puts the operation name in `meta["op"]`. It preserves the error's client-facing identity.
- `Boundary` preserves an `*errs.Error` identity when its item is not empty. It converts unknown or plain errors to `Internal` with a safe fallback item. This conversion includes structured wrappers around plain errors. The original error stays in the cause chain for logs.

Use `New`, `Newf`, `Specifyf`, `WithDetails`, and `Reason` as necessary. Use `B`, `Wrap`, `WrapCode`, and `Convert` when these conventions do not fit.

## Builder errors that convention helpers prevent

- `Cause()` copies the cause's `Item` into the builder. A later `Msg()` call has no effect. `Specify` sets the item before it attaches the cause.
- The builder drops `Details()` when the call occurs after `Cause()`. `WithDetails` sets details before it attaches the cause.
- `errs.B().Cause(plainErr).Err()` produces `Code: Unknown` and an empty item. `Boundary` gives these errors a safe identity.

## Extractors

```go
errs.Item(err)                  // stable token of the outermost *errs.Error
errs.IsCode(err, errs.NotFound) // what the edge turns into an HTTP status
errs.DetailsAs[MyDetails](err)  // typed details (in-process only)
```

## Typed details

```go
type SessionLimitDetails struct {
    Current int `json:"current"`
    Limit   int `json:"limit"`
}

func (SessionLimitDetails) ErrDetails() {}

err := errs.WithDetails(
    errs.Sentinel(errs.ResourceExhausted, "SESSION_LIMIT_EXCEEDED"),
    SessionLimitDetails{Current: 12, Limit: 10},
)
```

Use `errs.Reason{Message, Fields}` for free-form context.
