# Getting started with RonyKIT

Use this guide to build API servers with RonyKIT. You must know basic Go. No RonyKIT experience is necessary.

## Table of contents

- [Installation](#installation)
- [Your first server](#your-first-server)
- [Define routes and handlers](#define-routes-and-handlers)
- [Request binding](#request-binding)
- [State management](#state-management)
- [Middleware](#middleware)
- [Streaming handlers](#streaming-handlers)
- [Error handling](#error-handling)
- [API documentation](#api-documentation)
- [Client stubs](#client-stubs)
- [File uploads and raw bodies](#file-uploads-and-raw-bodies)
- [Testing](#testing)
- [Server options reference](#server-options-reference)
- [Context helpers reference](#context-helpers-reference)
- [Production checklist](#production-checklist)

## Installation

Install Go 1.25 or later. Then install the scaffolding tool:

```bash
go install github.com/clubpay/ronykit/ronyup@latest
```

Create a project with dependency injection, configuration, migrations, and repository layers:

```bash
ronyup setup workspace -r ./my-api -m github.com/you/my-api
cd my-api
ronyup setup feature -p users -n users -t service
```

These commands create a feature module at `feature/users/`.

You can also add the `rony` package directly to a Go project:

```bash
go get github.com/clubpay/ronykit/rony@latest
```

## Your first server

This example shows a complete server in one file:

```go
package main

import (
	"context"
	"os"

	"github.com/clubpay/ronykit/rony"
)

type GreetRequest struct {
	Name string `json:"name"`
}

type GreetResponse struct {
	Message string `json:"message"`
}

func Greet(ctx *rony.SUnaryCtx, in GreetRequest) (*GreetResponse, error) {
	return &GreetResponse{Message: "Hello, " + in.Name + "!"}, nil
}

func main() {
	srv := rony.NewServer(
		rony.WithServerName("greeter"),
		rony.WithVersion("v1.0.0"),
		rony.Listen(":8080"),
		rony.WithAPIDocs("/docs"),
		rony.UseSwaggerUI(),
	)

	rony.Setup(srv, "GreeterService", rony.EmptyState(),
		rony.WithUnary(Greet, rony.GET("/hello/{name}")),
	)

	_ = srv.Run(context.Background(), os.Interrupt, os.Kill)
}
```

Run the server and send a request:

```bash
go run main.go
curl http://localhost:8080/hello/World
# {"message":"Hello, World!"}
```

Open http://localhost:8080/docs to view the generated Swagger UI.

The example does these operations:

1. `rony.NewServer(...)` creates a server with an HTTP gateway on port 8080.
2. `rony.Setup(...)` registers one service and one contract. The contract binds `Greet` to `GET /hello/{name}`.
3. `srv.Run(...)` starts the server and waits for a shutdown signal.

## Define routes and handlers

### Unary handlers

A unary handler receives one typed request and returns one typed response:

```go
func GetUser(ctx *rony.SUnaryCtx, in GetUserRequest) (*GetUserResponse, error) {
// process request and return response
return &GetUserResponse{...}, nil
}
```

Use `rony.SUnaryCtx` for stateless handlers. Use `*rony.UnaryCtx[S, A]` for handlers that share state. See [State management](#state-management).

### Bind handlers to routes

Register handlers with route helpers in `rony.Setup`:

```go
rony.Setup(srv, "UserService", rony.EmptyState(),
	rony.WithUnary(CreateUser, rony.POST("/users")),
	rony.WithUnary(GetUser, rony.GET("/users/{id}")),
	rony.WithUnary(UpdateUser, rony.PUT("/users/{id}")),
	rony.WithUnary(DeleteUser, rony.DELETE("/users/{id}")),
	rony.WithUnary(ListUsers, rony.GET("/users")),
)
```

Available route helpers are `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`, and `ALL`. Use `REST(method, path)` for other HTTP methods.

### Route options

Use route options to set documentation metadata and behavior:

```go
rony.WithUnary(GetUser,
	rony.GET("/users/{id}", rony.UnaryName("GetUser")),
	rony.UnaryHeader(rony.RequiredHeader("Authorization")),
)
```

### Group related handlers

Use `SetupOptionGroup` to group handlers by domain:

```go
func userHandlers() rony.SetupOption[rony.EMPTY, rony.NOP] {
return rony.SetupOptionGroup(
rony.WithUnary(CreateUser, rony.POST("/users", rony.UnaryName("CreateUser"))),
rony.WithUnary(GetUser, rony.GET("/users/{id}", rony.UnaryName("GetUser"))),
rony.WithUnary(ListUsers, rony.GET("/users", rony.UnaryName("ListUsers"))),
)
}

func orderHandlers() rony.SetupOption[rony.EMPTY, rony.NOP] {
return rony.SetupOptionGroup(
rony.WithUnary(CreateOrder, rony.POST("/orders", rony.UnaryName("CreateOrder"))),
rony.WithUnary(GetOrder, rony.GET("/orders/{id}", rony.UnaryName("GetOrder"))),
)
}

rony.Setup(srv, "MyAPI", rony.EmptyState(),
userHandlers(),
orderHandlers(),
)
```

## Request binding

RonyKIT binds path parameters, query parameters, and request body fields to input struct fields. It uses the `json` tag to match fields.

```go
type SearchRequest struct {
	Category string `json:"category"` // from path /{category}, query ?category=, or body
	Query    string `json:"query"`    // from query ?query= or body
	Page     *int32 `json:"page"`     // optional - nil when not provided
}
```

Binding priority is path parameters, query parameters, then body fields.

Follow these rules:

- Use pointer types for optional fields. An omitted field stays `nil`.
- Use `ctx.GetInHdr("X-Request-Id")` to read request headers. RonyKIT does not bind headers to struct fields.
- Use the same tag names for path, query, and body fields.

## State management

Use the `rony` reducer pattern when handlers must share mutable data.

### Define the state

```go
type Counter struct {
	sync.Mutex
	Count int
}

func (c *Counter) Name() string { return "Counter" }
func (c *Counter) Reduce(action string) error {
	switch action {
	case "increment":
		c.Count++
	case "decrement":
		if c.Count <= 0 {
			return fmt.Errorf("count cannot go below zero")
		}
		c.Count--
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
	return nil
}
```

If the state implements `sync.Locker`, `ReduceState` locks the state during mutations.

### Use the state in a handler

```go
func Count(ctx *rony.UnaryCtx[*Counter, string], in CountRequest) (*CountResponse, error) {
	res := &CountResponse{}
	err := ctx.ReduceState(in.Action, func(s *Counter, err error) error {
		if err != nil {
			return err
		}
		res.Count = s.Count
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}
```

### Register the initial state

```go
rony.Setup(srv, "CounterService",
	rony.ToInitiateState[*Counter, string](&Counter{Count: 0}),
	rony.WithUnary(Count, rony.GET("/count/{action}")),
)
```

Use `rony.EmptyState()` and `rony.SUnaryCtx` when the service has no state.

Keep state small. Use dependency injection or handler closures for shared resources such as database pools and caches.

## Middleware

Middleware runs before handlers. Use middleware for authentication, logging, rate limiting, and request changes.

### Stateless middleware

Stateless middleware wraps `kit.HandlerFunc`:

```go
func LogMiddleware(ctx *kit.Context) {
	start := time.Now()
	ctx.Next()
	log.Printf("%s %s took %v", ctx.Route(), ctx.Conn().ClientIP(), time.Since(start))
}
```

Register middleware at the service level:

```go
rony.Setup(srv, "MyService", rony.EmptyState(),
	rony.WithMiddleware[rony.EMPTY, rony.NOP](LogMiddleware),
	rony.WithUnary(MyHandler, rony.GET("/items")),
)
```

Or register middleware at the contract level:

```go
rony.WithUnary(MyHandler,
	rony.GET("/items"),
	rony.UnaryMiddleware(AuthRequired),
)
```

Use stateless middleware unless the middleware must access shared state.

## Streaming handlers

Use streaming handlers for server push or bidirectional WebSocket messaging:

```go
type ChatIn struct {
Text string `json:"text"`
}

type ChatOut struct {
Text string `json:"text"`
}

func Chat(ctx *rony.SStreamCtx[ChatOut], in ChatIn) error {
ctx.Push(ChatOut{Text: "echo: " + in.Text})
return nil
}

rony.Setup(srv, "ChatService", rony.EmptyState(),
rony.WithStream(Chat, rony.RPC("chatMessage")),
)
```

- Use `ctx.Push(msg)` to send a message to the client.
- The `fasthttp` gateway usually carries streaming messages over WebSocket.

## Error handling

Return structured errors with the `rony/errs` package:

```go
import "github.com/clubpay/ronykit/rony/errs"

func GetUser(ctx *rony.SUnaryCtx, in GetUserRequest) (*GetUserResponse, error) {
	if in.ID == "" {
		return nil, errs.B().
			Code(errs.InvalidArgument).
			Msg("USER_ID_REQUIRED").
			Err()
	}

	user, err := userRepo.Get(ctx.Context(), in.ID)
	if err != nil {
		return nil, errs.B().
			Code(errs.Internal).
			Msg("FAILED_TO_GET_USER").
			Cause(err).
			Err()
	}

	return &GetUserResponse{User: user}, nil
}
```

RonyKIT maps error codes to HTTP status codes:

| Code | HTTP status |
|------|-------------|
| `errs.InvalidArgument` | 400 |
| `errs.Unauthenticated` | 401 |
| `errs.PermissionDenied` | 403 |
| `errs.NotFound` | 404 |
| `errs.AlreadyExists` | 409 |
| `errs.Internal` | 500 |

Use `WithErrorHandler` to replace global error serialization:

```go
rony.NewServer(
	rony.WithErrorHandler(func(ctx *kit.Context, err error) {
		ctx.SetStatusCode(400)
		ctx.Out().SetMsg(
			errs.B().Cause(err).Code(errs.InvalidArgument).Msg("COULD_NOT_PARSE_PAYLOAD").Err(),
		).Send()
	}),
)
```

## API documentation

RonyKIT generates OpenAPI documentation from handler types. It can serve this documentation at runtime.

### Serve documentation at runtime

```go
srv := rony.NewServer(
	rony.WithAPIDocs("/docs"),
	rony.UseSwaggerUI(), // or UseRedocUI() or UseScalarUI()
)
```

### Add metadata to documentation

```go
rony.WithUnary(Search,
	rony.GET("/search/{category}"),
	rony.UnaryInputMeta(
		desc.WithField("category", desc.FieldMeta{
			Description: "Product category",
			Enum:        []string{"electronics", "books", "clothing"},
		}),
	),
	rony.UnaryHeader(rony.RequiredHeader("Authorization")),
)
```

### Export documentation to a file

```go
_ = srv.GenDocFile(context.Background(), "openapi.json")
```

## Client stubs

Generate type-safe Go or TypeScript clients from service definitions.

### Go client

```go
import "github.com/clubpay/ronykit/stub/stubgen"

svcs := srv.ExportDesc()
stubgen.New(
stubgen.WithGenEngine(stubgen.NewGolangEngine(stubgen.GolangConfig{
PkgName: "myclient",
})),
stubgen.WithTags("json"),
stubgen.WithFolderName("myclient"),
stubgen.WithStubName("myService"),
).MustGenerate(svcs...)
```

### TypeScript client

```go
stubgen.New(
	stubgen.WithGenEngine(stubgen.NewTypescriptEngine(stubgen.TypescriptConfig{
		GenerateSWR: true,
	})),
	stubgen.WithTags("json"),
	stubgen.WithFolderName("myclient-ts"),
	stubgen.WithStubName("myService"),
).MustGenerate(svcs...)
```

## File uploads and raw bodies

### File uploads with multipart forms

```go
func Upload(ctx *rony.SUnaryCtx, in kit.MultipartFormMessage) (*kit.RawMessage, error) {
form := in.GetForm()
for name, files := range form.File {
_ = name
_ = files
}
empty := kit.RawMessage{}
return &empty, nil
}

rony.Setup(srv, "FileService", rony.EmptyState(),
rony.WithUnary(Upload, rony.POST("/upload")),
)
```

### Raw request and response bodies

```go
func RawEcho(ctx *rony.SUnaryCtx, in kit.RawMessage) (*kit.RawMessage, error) {
	return &in, nil
}
```

### HTTP redirects

```go
func Redirect(ctx *rony.SUnaryCtx, in RedirectRequest) (*rony.EMPTY, error) {
	if rc, ok := ctx.RESTConn(); ok {
		rc.Redirect(307, in.URL)
	}
	return nil, nil
}
```

## Testing

### Run the test suite

Run these commands from the repository root:

```bash
make setup    # Install tools; run one time
make test     # Run tests across all modules
make lint     # Lint all modules
```

Test one module with the Go tool:

```bash
cd rony && go test ./...
```

### Test handlers

Handlers are Go functions with typed inputs and outputs. Call them directly in unit tests:

```go
func TestGreet(t *testing.T) {
	resp, err := Greet(testCtx, GreetRequest{Name: "World"})
	assert.NoError(t, err)
	assert.Equal(t, "Hello, World!", resp.Message)
}
```

For integration tests, use generated Go client stubs to call endpoints.

## Server options reference

| Option | Purpose |
|--------|---------|
| `Listen(":8080")` | Set the HTTP gateway bind address |
| `WithServerName("name")` | Set the server name in documentation and logs |
| `WithVersion("v1.0.0")` | Set the API version |
| `WithCORS(config)` | Configure cross-origin resource sharing |
| `WithCompression(level)` | Enable gzip or zstd response compression |
| `WithAPIDocs("/docs")` | Serve OpenAPI documentation at the specified path |
| `UseSwaggerUI()` | Use Swagger UI instead of the default ReDoc |
| `UseScalarUI()` | Use Scalar UI for documentation |
| `WithTracer(tracer)` | Configure a distributed tracer |
| `WithLogger(logger)` | Configure a structured logger |
| `WithPrefork()` | Enable multi-process mode for higher throughput |
| `WithShutdownTimeout(d)` | Set the graceful shutdown timeout |
| `WithErrorHandler(fn)` | Replace global error serialization |
| `WithGlobalHandlers(h...)` | Add global middleware to all routes |

## Context helpers reference

| Method | Purpose |
|--------|---------|
| `ctx.GetInHdr("key")` | Read a request header |
| `ctx.SetOutHdr("key", "val")` | Set a response header |
| `ctx.Conn()` | Access the underlying connection |
| `ctx.Context()` | Get the request `context.Context` |
| `ctx.SetUserContext(c)` | Replace the request context |
| `ctx.RESTConn()` | Access REST helpers for method, path, and redirect |
| `ctx.StopExecution()` | Stop the remaining middleware |
| `ctx.KitCtx()` | Access the underlying `kit.Context` |

## Production checklist

- [ ] Use consistent `rony/errs` error codes in all handlers.
- [ ] Add correct tags to DTOs. Use pointer types for optional fields.
- [ ] Add field metadata, enums, and descriptions to API documentation.
- [ ] Configure CORS with `rony.WithCORS(rony.CORSConfig{...})`.
- [ ] Enable compression with `rony.WithCompression(...)`.
- [ ] Add authentication middleware to protected endpoints.
- [ ] Configure tracing and logging with `rony.WithTracer` and `rony.WithLogger`.
- [ ] Configure reverse proxy and service timeouts. Use `rony.WithShutdownTimeout` for shutdown.
- [ ] Generate client stubs and distribute them to consumers.
- [ ] Run `make test` and `make lint`. Both commands must pass.

## Next steps

- [Cookbook](./cookbook.md): Use patterns for authentication, pagination, and validation.
- [ronyup Guide](./ronyup-guide.md): Use the scaffolding CLI and MCP server.
- [Architecture](./architecture.md): Learn about internal components.
- [Advanced: Kit](./advanced-kit.md): Build custom gateways with low-level tools.
