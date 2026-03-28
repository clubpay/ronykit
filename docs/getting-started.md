# Getting Started with RonyKIT

This guide walks you through building API servers with RonyKIT. No prior RonyKIT
experience required — basic Go knowledge is enough.

## Table of Contents

- [Installation](#installation)
- [Your First Server](#your-first-server)
- [Defining Routes and Handlers](#defining-routes-and-handlers)
- [Request Binding](#request-binding)
- [State Management](#state-management)
- [Middleware](#middleware)
- [Streaming Handlers](#streaming-handlers)
- [Error Handling](#error-handling)
- [API Docs](#api-docs)
- [Client Stubs](#client-stubs)
- [File Uploads and Raw Bodies](#file-uploads-and-raw-bodies)
- [Testing](#testing)
- [Server Options Reference](#server-options-reference)
- [Context Helpers Reference](#context-helpers-reference)
- [Production Checklist](#production-checklist)

---

## Installation

You need **Go 1.25+**. Install the scaffolding tool:

```bash
go install github.com/clubpay/ronykit/ronyup@latest
```

To scaffold a complete project with DI, config, migrations, and repo layers:

```bash
ronyup setup workspace -r ./my-api -m github.com/you/my-api
cd my-api
ronyup setup feature -p users -n users -t service
```

Or start from scratch by adding the `rony` package directly:

```bash
go get github.com/clubpay/ronykit/rony@latest
```

---

## Your First Server

A complete, working server in a single file:

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

Run it and try it:

```bash
go run main.go
curl http://localhost:8080/hello/World
# {"message":"Hello, World!"}
```

Open http://localhost:8080/docs for auto-generated Swagger UI.

**What happened:**

1. `rony.NewServer(...)` creates a server with an HTTP gateway on port 8080.
2. `rony.Setup(...)` registers a service with one contract (the `Greet` handler bound to `GET /hello/{name}`).
3. `srv.Run(...)` starts the server and blocks until a shutdown signal arrives.

---

## Defining Routes and Handlers

### Unary handlers (request -> response)

The most common handler type. Takes a typed request, returns a typed response:

```go
func GetUser(ctx *rony.SUnaryCtx, in GetUserRequest) (*GetUserResponse, error) {
// process request and return response
return &GetUserResponse{...}, nil
}
```

`rony.SUnaryCtx` is a shorthand for stateless handlers. If you need shared state,
use `*rony.UnaryCtx[S, A]` (see [State Management](#state-management)).

### Binding handlers to routes

Register handlers with route helpers during `rony.Setup`:

```go
rony.Setup(srv, "UserService", rony.EmptyState(),
rony.WithUnary(CreateUser, rony.POST("/users")),
rony.WithUnary(GetUser, rony.GET("/users/{id}")),
rony.WithUnary(UpdateUser, rony.PUT("/users/{id}")),
rony.WithUnary(DeleteUser, rony.DELETE("/users/{id}")),
rony.WithUnary(ListUsers, rony.GET("/users")),
)
```

Available route helpers: `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`,
`ALL`, and the generic `REST(method, path)`.

### Route options

Customize individual routes for documentation and behavior:

```go
rony.WithUnary(GetUser,
rony.GET("/users/{id}", rony.UnaryName("GetUser")),
rony.UnaryHeader(rony.RequiredHeader("Authorization")),
)
```

### Grouping related handlers

Use `SetupOptionGroup` to organize handlers by domain:

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

---

## Request Binding

RonyKIT automatically binds path parameters, query parameters, and request body
to your input struct fields. It matches fields using the `json` tag.

```go
type SearchRequest struct {
Category string `json:"category"` // from path /{category}, query ?category=, or body
Query    string `json:"query"`    // from query ?query= or body
Page     *int32 `json:"page"` // optional — nil when not provided
}
```

**Priority:** path params > query params > body fields.

**Tips:**

- Use **pointer types** for optional fields — they stay `nil` when not provided.
- Read request headers with `ctx.GetInHdr("X-Request-Id")` — headers aren't bound to struct fields.
- Keep tag names consistent across path, query, and body.

---

## State Management

If handlers need to share mutable data, `rony` provides a built-in reducer pattern.

### Define your state

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

When your state implements `sync.Locker`, `ReduceState` automatically locks around mutations.

### Use it in a handler

```go
func Count(ctx *rony.UnaryCtx[*Counter, string], in CountRequest) (*CountResponse, error) {
res := &CountResponse{}
err := ctx.ReduceState(in.Action, func (s *Counter, err error) error {
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

### Register with initial state

```go
rony.Setup(srv, "CounterService",
rony.ToInitiateState[*Counter, string](&Counter{Count: 0}),
rony.WithUnary(Count, rony.GET("/count/{action}")),
)
```

When you don't need state, use `rony.EmptyState()` and the `rony.SUnaryCtx` shorthand.

**Advice:** Keep state small. For heavy shared resources (database pools, caches),
use dependency injection or pass them through handler closures.

---

## Middleware

Middleware runs before your handlers. Use it for auth, logging, rate limiting,
or request shaping.

### Stateless middleware

The most common kind — wraps `kit.HandlerFunc`:

```go
func LogMiddleware(ctx *kit.Context) {
start := time.Now()
ctx.Next()
log.Printf("%s %s took %v", ctx.Route(), ctx.Conn().ClientIP(), time.Since(start))
}
```

Register at service level:

```go
rony.Setup(srv, "MyService", rony.EmptyState(),
rony.WithMiddleware[rony.EMPTY, rony.NOP](LogMiddleware),
rony.WithUnary(MyHandler, rony.GET("/items")),
)
```

Or at contract level:

```go
rony.WithUnary(MyHandler,
rony.GET("/items"),
rony.UnaryMiddleware(AuthRequired),
)
```

**Prefer stateless middleware** unless you truly need shared state access.

---

## Streaming Handlers

For server-push or bidirectional messaging (WebSocket), use streaming handlers:

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

- Use `ctx.Push(msg)` to send messages to the client.
- Streaming is typically used over WebSocket via the `fasthttp` gateway.

---

## Error Handling

Return structured errors using the `rony/errs` package:

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

Error codes map to HTTP status codes automatically:

| Code                    | HTTP Status |
|-------------------------|-------------|
| `errs.InvalidArgument`  | 400         |
| `errs.Unauthenticated`  | 401         |
| `errs.PermissionDenied` | 403         |
| `errs.NotFound`         | 404         |
| `errs.AlreadyExists`    | 409         |
| `errs.Internal`         | 500         |

Override error serialization globally:

```go
rony.NewServer(
rony.WithErrorHandler(func (ctx *kit.Context, err error) {
ctx.SetStatusCode(400)
ctx.Out().SetMsg(
errs.B().Cause(err).Code(errs.InvalidArgument).Msg("COULD_NOT_PARSE_PAYLOAD").Err(),
).Send()
}),
)
```

---

## API Docs

RonyKIT generates and serves OpenAPI documentation from your handler types.

### Serve docs at runtime

```go
srv := rony.NewServer(
rony.WithAPIDocs("/docs"),
rony.UseSwaggerUI(), // or UseRedocUI() or UseScalarUI()
)
```

### Enrich docs with metadata

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

### Export to file

```go
_ = srv.GenDocFile(context.Background(), "openapi.json")
```

---

## Client Stubs

Generate type-safe clients in Go or TypeScript from your service definitions.

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

---

## File Uploads and Raw Bodies

### File uploads (multipart forms)

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

### Raw request/response bodies

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

---

## Testing

### Run the test suite

```bash
make setup    # install tools — run once
make test     # run tests across all modules
make lint     # lint all modules
```

To test a single module:

```bash
cd rony && go test ./...
```

### Test your handlers

Handlers are regular Go functions with typed inputs and outputs — test them directly:

```go
func TestGreet(t *testing.T) {
resp, err := Greet(testCtx, GreetRequest{Name: "World"})
assert.NoError(t, err)
assert.Equal(t, "Hello, World!", resp.Message)
}
```

For integration tests, use generated Go client stubs to call your endpoints end-to-end.

---

## Server Options Reference

| Option                     | Description                                 |
|----------------------------|---------------------------------------------|
| `Listen(":8080")`          | Bind address for the HTTP gateway           |
| `WithServerName("name")`   | Server name (appears in docs and logs)      |
| `WithVersion("v1.0.0")`    | API version                                 |
| `WithCORS(config)`         | Cross-origin resource sharing               |
| `WithCompression(level)`   | Response compression (gzip/zstd)            |
| `WithAPIDocs("/docs")`     | Serve OpenAPI docs at the given path        |
| `UseSwaggerUI()`           | Use Swagger UI instead of the default ReDoc |
| `UseScalarUI()`            | Use Scalar UI for docs                      |
| `WithTracer(tracer)`       | Plug in a distributed tracer                |
| `WithLogger(logger)`       | Plug in a structured logger                 |
| `WithPrefork()`            | Multi-process mode for higher throughput    |
| `WithShutdownTimeout(d)`   | Graceful shutdown timeout                   |
| `WithErrorHandler(fn)`     | Override global error serialization         |
| `WithGlobalHandlers(h...)` | Global middleware for all routes            |

---

## Context Helpers Reference

| Method                        | Description                                           |
|-------------------------------|-------------------------------------------------------|
| `ctx.GetInHdr("key")`         | Read a request header                                 |
| `ctx.SetOutHdr("key", "val")` | Set a response header                                 |
| `ctx.Conn()`                  | Access the underlying connection                      |
| `ctx.Context()`               | Get the `context.Context` for the request             |
| `ctx.SetUserContext(c)`       | Replace the request context                           |
| `ctx.RESTConn()`              | Access REST-specific helpers (method, path, redirect) |
| `ctx.StopExecution()`         | Short-circuit remaining middleware                    |
| `ctx.KitCtx()`                | Access the underlying `kit.Context`                   |

---

## Production Checklist

- [ ] **Error handling** — consistent error codes from `rony/errs` across all handlers
- [ ] **Input validation** — DTOs properly tagged; optional fields use pointer types
- [ ] **API docs** — enriched with field metadata, enums, and descriptions
- [ ] **CORS** — configured with `rony.WithCORS(rony.CORSConfig{...})`
- [ ] **Compression** — enabled with `rony.WithCompression(...)`
- [ ] **Auth middleware** — in place for protected endpoints
- [ ] **Observability** — tracer and logger wired with `rony.WithTracer` / `rony.WithLogger`
- [ ] **Timeouts** — set at reverse proxy and service layer with `rony.WithShutdownTimeout`
- [ ] **Client stubs** — generated and distributed to consumers
- [ ] **Tests** — `make test` and `make lint` both green

---

## Next Steps

- [Cookbook](./cookbook.md) — production patterns for auth, pagination, validation, and more
- [ronyup Guide](./ronyup-guide.md) — scaffolding CLI and MCP server
- [Architecture](./architecture.md) — how RonyKit works internally
- [Advanced: Kit](./advanced-kit.md) — low-level toolkit for custom gateways
