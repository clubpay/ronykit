---
name: design-api
description: Guide an AI agent through designing RonyKIT APIs with contracts, routes, handlers, and documentation.
arguments:
  - name: service_name
    description: The service module name (without "mod" suffix).
    required: true
  - name: endpoints
    description: "Comma-separated list of endpoint names or brief descriptions (e.g. 'CreateUser, GetUser, ListUsers, DeleteUser')."
    required: true
  - name: auth_required
    description: "Whether endpoints require authentication (yes/no)."
    required: false
---
You are designing the API surface for the "{{service_name}}mod" RonyKIT service.

Requested endpoints: {{endpoints}}

{{#if auth_required}}
Authentication required: {{auth_required}}
{{/if}}

## API Design Principles

1. **Contracts = single API operations**: each endpoint is a Contract with typed input/output, route selector, and handler chain.
2. **Thin handlers**: handlers validate input, delegate to `internal/app`, and map results to output DTOs.
3. **Domain types stay internal**: API input/output DTOs use `json` and `swag` tags. Domain types in `internal/domain` have NO json tags.
4. **Separation of concerns**: API layer handles serialization/validation, app layer owns business logic, repo layer owns persistence.

## Defining Contracts with rony.Setup

```go
func (svc Service) Desc() *desc.Service {
    return rony.Setup[*State, Action](
        "{{service_name}}",
        rony.ToInitiateState[*State, Action](&State{}),

        // REST endpoints
        rony.WithUnary(svc.CreateItem,
            rony.POST("/v1/{{service_name}}/items"),
        ),
        rony.WithUnary(svc.GetItem,
            rony.GET("/v1/{{service_name}}/items/{id}"),
        ),
        rony.WithUnary(svc.ListItems,
            rony.GET("/v1/{{service_name}}/items"),
        ),
        rony.WithUnary(svc.UpdateItem,
            rony.PUT("/v1/{{service_name}}/items/{id}"),
        ),
        rony.WithUnary(svc.DeleteItem,
            rony.DELETE("/v1/{{service_name}}/items/{id}"),
        ),

        // WebSocket/RPC endpoints (if needed)
        rony.WithStream(svc.WatchItems,
            rony.RPC("{{service_name}}.WatchItems"),
        ),

        // Service-level middleware
        rony.WithMiddleware[*State, Action](authMiddleware),
    )
}
```

## Route Selectors

| Helper | HTTP Method | Example |
|--------|------------|---------|
| `rony.GET(path)` | GET | `rony.GET("/v1/users/{id}")` |
| `rony.POST(path)` | POST | `rony.POST("/v1/users")` |
| `rony.PUT(path)` | PUT | `rony.PUT("/v1/users/{id}")` |
| `rony.PATCH(path)` | PATCH | `rony.PATCH("/v1/users/{id}")` |
| `rony.DELETE(path)` | DELETE | `rony.DELETE("/v1/users/{id}")` |
| `rony.REST(method, path)` | Any | `rony.REST("OPTIONS", "/v1/health")` |
| `rony.RPC(predicate)` | WebSocket/RPC | `rony.RPC("users.Subscribe")` |

Path parameters use `{name}` syntax. Query parameters are extracted from input struct fields.

## Input/Output DTO Patterns

```go
type CreateItemInput struct {
    Name        string `json:"name"        swag:"required,The item name"`
    Description string `json:"description" swag:"The item description"`
    CategoryID  int64  `json:"categoryId"  swag:"required,Category identifier"`
}

type CreateItemOutput struct {
    ID        int64  `json:"id"`
    Name      string `json:"name"`
    CreatedAt int64  `json:"createdAt"`
}
```

For path parameters, use the same field name as in the route:

```go
type GetItemInput struct {
    ID int64 `json:"id" swag:"required,The item ID"`
}
```

For list endpoints with pagination:

```go
type ListItemsInput struct {
    Offset int32  `json:"offset" swag:"Pagination offset"`
    Limit  int32  `json:"limit"  swag:"Pagination limit (default 20)"`
    Sort   string `json:"sort"   swag:"Sort field"`
}

type ListItemsOutput struct {
    Items      []ItemSummary `json:"items"`
    TotalCount int64         `json:"totalCount"`
}
```

## Handler Implementation

```go
func (svc Service) CreateItem(ctx *RContext, in CreateItemInput) (*CreateItemOutput, error) {
    item, err := svc.app.CreateItem(ctx.Context(), app.CreateItemParams{
        Name:        in.Name,
        Description: in.Description,
        CategoryID:  in.CategoryID,
    })
    if err != nil {
        return nil, err
    }

    return toCreateItemOutput(item), nil
}
```

## Enum Documentation on Input Fields

Use `rony.UnaryInputMeta` with `desc.WithField` to document enum values:

```go
rony.WithUnary(svc.ListItems,
    rony.GET("/v1/{{service_name}}/items"),
    rony.UnaryInputMeta(
        desc.WithField("sort", "name", "created_at", "updated_at"),
        desc.WithField("status", "active", "archived"),
    ),
),
```

## Error Responses

Return errors from `rony/errs` — the framework maps them to HTTP status codes automatically:

| errs Code | HTTP Status |
|-----------|-------------|
| `errs.InvalidArgument` | 400 |
| `errs.Unauthenticated` | 401 |
| `errs.PermissionDenied` | 403 |
| `errs.NotFound` | 404 |
| `errs.AlreadyExists` | 409 |
| `errs.FailedPrecondition` | 412 |
| `errs.Internal` | 500 |

## API Documentation

Enable built-in API docs on the server:

```go
srv := rony.NewServer(
    rony.Listen(":8080"),
    rony.WithAPIDocs("/docs"),
    rony.UseScalarUI(),  // or UseSwaggerUI(), UseRedocUI()
)
```

Generate OpenAPI spec with `x/apidoc` from service descriptors.

## File Organization

- `api/service.go` — RContext alias, ServiceParams, Service struct, Desc() method
- `api/api_<domain>.go` — Per-domain: Input/Output DTOs at top, handler methods, mapping helpers (toOutputDTO)
- Keep related CRUD endpoints in the same `api_<domain>.go` file
- One handler file per logical domain concept (accounts, transfers, policies, etc.)

## Checklist

1. Define route paths following REST conventions (nouns, not verbs).
2. Create Input/Output DTOs with proper `json` and `swag` tags.
3. Write thin handler methods that delegate to app layer.
4. Add `rony.UnaryInputMeta` for enum fields.
5. Define domain errors in `internal/domain/errors.go`.
6. Enable API docs with `rony.WithAPIDocs`.
7. Run `make gen-stub` after finalizing contracts.
