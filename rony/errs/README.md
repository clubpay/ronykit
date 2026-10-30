# rony/errs — structured errors and service conventions

`rony/errs` provides the error mechanics used by RonyKit (the `Error` type, builder, gRPC-style codes, wrapping, HTTP mapping, and serialization) together with optional conventions for a stable client-facing error contract.

## Wire contract

Every error crossing an API boundary can be represented as an `*errs.Error`. Its REST body is:

```json
{"code": 3, "codeName": "invalid_argument", "item": "TRIGGER_GITHUB_ORG_REQUIRED", "details": null}
```

| Field      | Role                                                                                                                        |
|------------|-----------------------------------------------------------------------------------------------------------------------------|
| `code`     | gRPC-style code; the edge derives the HTTP status from it. Never invent custom codes.                                       |
| `codeName` | The code's string form.                                                                                                     |
| `item`     | Stable, machine-readable token, SCREAMING_SNAKE, namespaced by area. Clients may branch on it; never rename a shipped item. |
| `details`  | Client-visible structured context (`Reason` or any `errs.ErrDetails` struct).                                               |
| `meta`     | Internal diagnostics; not exposed to external clients, though trusted internal serialization may propagate it.              |

The **cause chain is not serialized to external clients**: only the boundary error's `item` reaches them. Everything a client needs must be in `item`/`details`; everything an operator needs can stay in the cause chain and `meta`.

## Layer conventions

```go
// Domain: sentinels once, specific failures everywhere.
var ErrInvalidArgument = errs.Sentinel(errs.InvalidArgument, "INVALID_ARGUMENT")

if spec.Org == "" {
	return errs.Specify(ErrInvalidArgument, "TRIGGER_GITHUB_ORG_REQUIRED")
}

// App: annotate, don't rebuild. Wire identity is untouched.
return errs.Annotate(err, "automation.Create")

// API: normalize at the boundary.
return errs.Boundary(err, "AUTOMATION_INTERNAL")
```

- `Specify` keeps the sentinel on the cause chain, so `errors.Is` keeps working, while the error carries a specific item for clients.
- `Annotate` puts the operation name in `meta["op"]` and preserves the error's client-facing identity.
- `Boundary` preserves an `*errs.Error` identity with a non-empty item and escapes unknown/plain errors—including structured shells around plain errors—to `Internal` with a safe fallback item. The original error stays on the cause chain for logs.

Use `New`, `Newf`, `Specifyf`, `WithDetails`, and `Reason` as needed. The lower-level `B`, `Wrap`, `WrapCode`, and `Convert` APIs remain available when these conventions do not fit.

## Builder footguns the convention helpers avoid

- `Cause()` inherits the cause's `Item` into the builder, so a `Msg()` chained **after** `Cause()` is a silent no-op. `Specify` sets the item before attaching the cause.
- `Details()` chained **after** `Cause()` is silently dropped. `WithDetails` sets details before attaching the cause.
- `errs.B().Cause(plainErr).Err()` yields `Code: Unknown` and an empty item. `Boundary` gives such errors a safe identity.

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

For free-form context use `errs.Reason{Message, Fields}`.
