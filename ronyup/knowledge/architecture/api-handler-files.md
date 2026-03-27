Organize API handlers into separate files per domain concept:

- `api/service.go` for route registration and the `Service` struct,
- `api/api_account.go`, `api/api_jwt.go`, `api/api_password.go`, etc. for
  handler implementations.

Each `api_*.go` file defines its input/output DTOs at the top (with `json` and
`swag` tags), followed by handler methods.

Handler methods use:

- `func (svc Service) MethodName(ctx *RContext, input InputType) (*OutputType, error)`

Keep handlers thin:

- validate input,
- call the corresponding app method,
- map the app result to the API output type.

Use helper functions (`toAccount`, `toWalletAccount`) for domain-to-API type
conversion within the same file.

Use `rony.UnaryInputMeta` with `desc.WithField` for enum documentation on input
fields.
