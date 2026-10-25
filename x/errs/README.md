# x/errs — opinionated error dialect on top of rony/errs

`rony/errs` gives you the mechanics (builder, gRPC-style codes, `Convert`,
`Details`/`Meta`). `x/errs` adds the conventions that keep a fleet of
services consistent, plus helpers that avoid the builder's footguns.

## Wire contract

Every error crossing an API boundary is an `*errs.Error`. Its REST body is:

```json
{"code": 3, "codeName": "invalid_argument", "item": "TRIGGER_GITHUB_ORG_REQUIRED", "details": null}
```

| Field     | Role                                                                                       |
|-----------|--------------------------------------------------------------------------------------------|
| `code`    | gRPC-style code; the edge derives the HTTP status from it. Never invent custom codes.      |
| `codeName`| The code's string form, filled at construction (since rony v0.27.4).                       |
| `item`    | Stable, machine-readable token, SCREAMING_SNAKE, namespaced by area. Clients may branch on it — never rename a shipped item. |
| `details` | Client-visible structured context (`Reason` or any `errs.ErrDetails` struct).              |
| `meta`    | Internal only (`json:"-"`); never leaves the process.                                      |

The **cause chain is not serialized**: only the boundary error's `item`
reaches clients. Everything a client needs must be in `item`/`details`;
everything an operator needs can stay in the cause chain and `meta`.

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

- `Specify` keeps the sentinel on the cause chain, so `errors.Is` keeps
  working, while the error carries a specific item for clients.
- `Annotate` puts the operation name in `meta["op"]` — it changes nothing
  on the wire, unlike rebuilding with a generic message (which discards
  the specific item clients need).
- `Boundary` preserves the richest `*errs.Error` identity in the chain and
  escapes unknown/plain errors to `Internal` with a safe fallback item,
  keeping the original on the cause chain for logs.

## Builder footguns this package avoids

- `Cause()` inherits the cause's `Item` into the builder, so a `Msg()`
  chained **after** `Cause()` is a silent no-op. `Specify` sets the item
  before attaching the cause.
- `Details()` chained **after** `Cause()` is silently dropped. `WithDetails`
  sets it before attaching the cause.
- `errs.B().Cause(plainErr).Err()` yields `Code: Unknown` and an empty
  item. `Boundary` gives such errors a safe identity.

## Extractors

```go
errs.Item(err)                    // stable token of the outermost *errs.Error
errs.IsCode(err, errs.NotFound)   // what the edge turns into an HTTP status
errs.DetailsAs[MyDetails](err)    // typed details (in-process only)
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
