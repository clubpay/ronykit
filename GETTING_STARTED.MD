# Getting Started with RonyKIT

This guide walks you through building API servers with **RonyKIT** — from your first
"Hello World" endpoint to production-ready patterns. No prior RonyKIT experience is
required; basic Go knowledge is enough.

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Your First Server](#your-first-server)
- [rony vs kit — Choosing the Right Level](#rony-vs-kit--choosing-the-right-level)
- [Defining Routes and Handlers](#defining-routes-and-handlers)
- [Request Binding: Path, Query, and Body](#request-binding-path-query-and-body)
- [State Management](#state-management)
- [Middleware](#middleware)
- [Streaming (RPC) Handlers](#streaming-rpc-handlers)
- [Error Handling](#error-handling)
- [Generating API Docs](#generating-api-docs)
- [Generating Client Stubs](#generating-client-stubs)
- [File Uploads and Raw Bodies](#file-uploads-and-raw-bodies)
- [Testing](#testing)
- [Production Checklist](#production-checklist)

---

## Overview

RonyKIT has two layers:

| Layer          | Package | Best for                                                      |
|----------------|---------|---------------------------------------------------------------|
| **High-level** | `rony`  | Most services — easy setup, type-safe handlers, built-in docs |
| **Low-level**  | `kit`   | Custom gateways, protocols, or deeply customized routing      |

**Start with `rony`.** You can always drop down to `kit` later if you need more control.

---

## Installation

You need **Go 1.25+** installed. Then grab the scaffolding tool:

```bash
go install github.com/clubpay/ronykit/ronyup@latest
```

To scaffold a new project:

```bash
ronyup setup -d ./my-service -m github.com/you/my-service -p MyService
```

This generates a ready-to-run project with dependency wiring, a gateway module, and
a place to add your API handlers. You can also skip scaffolding and start from scratch
as shown below.

---

## Your First Server

Here is a complete, working server in a single file. It defines one endpoint that
greets the caller by name.

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
		rony.WithUnary(Greet, rony.GET("/hello/:name")),
	)

	_ = srv.Run(context.Background(), os.Interrupt, os.Kill)
}
```

Run it:

```bash
go run main.go
```

Try it:

```bash
curl http://localhost:8080/hello/World
# → {"message":"Hello, World!"}
```

Open http://localhost:8080/docs to see the auto-generated Swagger UI.

**What just happened?**

1. `rony.NewServer(...)` creates a server with an HTTP gateway listening on port 8080.
2. `rony.Setup(...)` registers a **service** called "GreeterService" with one **contract**
	 (the `Greet` handler bound to `GET /hello/:name`).
3. `srv.Run(...)` starts the server and blocks until it receives a shutdown signal.

---

## rony vs kit — Choosing the Right Level

|                      | `rony`                               | `kit`                                                    |
|----------------------|--------------------------------------|----------------------------------------------------------|
| **Handler style**    | Typed: `func(ctx, In) (*Out, error)` | Raw: `func(ctx *kit.Context)`                            |
| **State management** | Built-in reducer pattern             | Manual                                                   |
| **API docs**         | Auto-generated from handler types    | Build with `kit/desc` manually                           |
| **Setup**            | `rony.NewServer` + `rony.Setup`      | Wire `kit.EdgeServer`, gateway, and descriptors yourself |
| **When to use**      | Most services                        | Custom gateways, non-HTTP protocols, or advanced routing |

**Key types in `rony`:**

- **`rony.Server`** — wraps a `kit.EdgeServer`; manages lifecycle and optional docs serving.
- **`rony.Setup`** — registers services, contracts, and handlers with type-safe signatures.
- **`rony.UnaryHandler`** — a handler that receives one request and returns one response.
- **`rony.StreamHandler`** — a handler for server-push or bidirectional patterns.
- **`rony.UnaryCtx` / `rony.StreamCtx`** — request-scoped context with helpers for headers, connections, etc.
- **`rony.State[A]`** — optional shared state with a reducer pattern (more on this later).

---

## Defining Routes and Handlers

### Unary handlers (request → response)

A unary handler takes a typed request and returns a typed response:

```go
func MyHandler(ctx *rony.SUnaryCtx, in MyRequest) (*MyResponse, error) {
	// Process the request and return a response
	return &MyResponse{...}, nil
}
```

`rony.SUnaryCtx` is a shorthand for handlers that don't use shared state. If you need
state, use `*rony.UnaryCtx[S, A]` instead (see [State Management](#state-management)).

### Binding handlers to routes

Register handlers during `rony.Setup` using route helpers:

```go
rony.Setup(srv, "MyService", rony.EmptyState(),
	rony.WithUnary(MyHandler,
		rony.GET("/items/:id"),             // GET with a path parameter
		rony.POST("/items"),                // POST to the same handler
	),
	rony.WithUnary(AnotherHandler,
		rony.PUT("/items/:id"),
	),
)
```

Available route helpers: `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`,
and the generic `REST(method, path)`.

### Route options

You can customize individual routes:

```go
rony.WithUnary(MyHandler,
	rony.GET("/items/:id",
		rony.UnaryName("GetItem"),          // custom name (appears in docs)
		rony.UnaryDeprecated(true),         // mark as deprecated
	),
	rony.UnaryHeader("Authorization", true), // declare a required header
)
```

### Multiple routes, one handler

A single handler can serve multiple routes. Give each route a unique name so they
appear separately in the generated docs:

```go
rony.WithUnary(Echo,
	rony.GET("/echo/:id", rony.UnaryName("EchoGET")),
	rony.POST("/echo", rony.UnaryName("EchoPOST")),
)
```

---

## Request Binding: Path, Query, and Body

When a request arrives, RonyKIT automatically fills your input struct from path
parameters, query parameters, and the request body. It matches fields using the
struct tag that corresponds to the service encoding (default: `json`).

```go
type SearchRequest struct {
	Category string `json:"category"` // from path /:category, query ?category=, or body
	Query    string `json:"query"`    // from query ?query= or body
	Page     *int32 `json:"page"`     // optional — pointer fields are only set when provided
}
```

**Tips:**

- Path parameters (`:category`) take the highest priority, then query, then body.
- Use **pointer types** for truly optional fields — they stay `nil` when not provided.
- Keep tag names simple and consistent. They're used across path, query, and body.
- To read request headers, use `ctx.GetInHdr("X-Request-Id")` — headers aren't bound
	to struct fields.

---

## State Management

If your handlers need to share mutable data (like an in-memory counter), `rony`
provides a built-in **reducer pattern**.

### Step 1: Define your state

Your state must implement the `rony.State` interface — a `Name()` method and a
`Reduce(action)` method:

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

Embedding `sync.Mutex` is recommended. When your state implements `sync.Locker`,
`ReduceState` automatically locks around mutations.

### Step 2: Use it in a handler

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

### Step 3: Register with initial state

```go
rony.Setup(srv, "CounterService",
	rony.ToInitiateState[*Counter, string](&Counter{Count: 0}),
	rony.WithUnary(Count, rony.GET("/count/:action")),
)
```

**When you don't need state**, use `rony.EmptyState()` and the `rony.SUnaryCtx`
shorthand (as shown in the earlier examples).

**Advice:** Keep state small. For heavy shared resources (database pools, caches),
use dependency injection or pass them through handler closures instead.

---

## Middleware

Middleware runs before your handlers. Use it for auth checks, logging, rate limiting,
or request shaping.

### Stateless middleware

The most common kind — a simple wrapper around `kit.HandlerFunc`:

```go
func AuthMiddleware(next kit.HandlerFunc) kit.HandlerFunc {
	return func(kc *kit.Context) {
		token := kc.In().GetHdr("Authorization")
		if token == "" {
			kc.Error(errs.B().Code(errs.PermissionDenied).Msg("missing auth token").Err())
			return
		}
		next(kc)
	}
}
```

Register it during setup:

```go
rony.Setup(srv, "SecuredService", rony.EmptyState(),
	rony.WithMiddleware[rony.EMPTY, rony.NOP](AuthMiddleware),
	rony.WithUnary(SecretHandler, rony.GET("/secret")),
)
```

### Stateful middleware

If your middleware needs to access shared state, use `*rony.BaseCtx[S, A]`:

```go
func HitCounter[S rony.State[A], A rony.Action](ctx *rony.BaseCtx[S, A]) {
	_ = ctx.ReduceState(any("hit").(A), nil)
}

rony.Setup(srv, "Tracked", rony.ToInitiateState(&MyState{}),
	rony.WithMiddleware[*MyState, string](HitCounter[*MyState, string]),
	rony.WithUnary(MyHandler, rony.GET("/tracked")),
)
```

**Prefer stateless middleware** unless you truly need shared state access.

---

## Streaming (RPC) Handlers

For server-push or bidirectional messaging (e.g., chat, live updates), use streaming
handlers with `rony.WithStream` and `RPC(predicate)`:

```go
type ChatIn struct {
	Text string `json:"text"`
}
type ChatOut struct {
	Text string `json:"text"`
}

func Chat(ctx *rony.StreamCtx[*MyState, string, ChatOut], in ChatIn) error {
	ctx.Push(ChatOut{Text: "echo: " + in.Text})
	return nil
}

rony.Setup(srv, "ChatService", rony.ToInitiateState(&MyState{}),
	rony.WithStream(Chat, rony.RPC("chatMessage")),
)
```

- Use `ctx.Push(msg)` to send messages to the client.
- You can add headers to pushed messages: `ctx.Push(msg, rony.WithHdr("key", "value"))`.
- Streaming is typically used over WebSocket connections via the `fasthttp` gateway.

---

## Error Handling

Handlers return `(*OUT, error)`. Use the `rony/errs` package to return structured
errors with consistent codes.

### Building errors

```go
import "github.com/clubpay/ronykit/rony/errs"

func MyHandler(ctx *rony.SUnaryCtx, in MyRequest) (*MyResponse, error) {
	if in.Name == "" {
		return nil, errs.B().
			Code(errs.InvalidArgument).
			Msg("name is required").
			Err()
	}
	return &MyResponse{}, nil
}
```

### Wrapping existing errors

```go
result, err := db.Query(...)
if err != nil {
	return nil, errs.WrapCode(err, errs.Internal, "database query failed")
}
```

### Custom error handling

Override how errors are serialized globally:

```go
rony.NewServer(
	rony.WithErrorHandler(func(kc *kit.Context, err error) {
		// Write your custom error envelope here
	}),
)
```

Common error codes: `errs.InvalidArgument`, `errs.PermissionDenied`, `errs.NotFound`,
`errs.Internal`. These map to appropriate HTTP status codes automatically.

---

## Generating API Docs

RonyKIT can generate and serve OpenAPI documentation directly from your handler
definitions — no separate spec files needed.

### Serve docs at runtime

```go
srv := rony.NewServer(
	rony.WithAPIDocs("/docs"),    // serve docs at /docs
	rony.UseSwaggerUI(),          // use Swagger UI (default is ReDoc)
	// ...
)
```

### Enrich docs with metadata

Add field descriptions, enums, and header requirements to make docs more useful:

```go
rony.WithUnary(Search,
	rony.GET("/search/:category"),
	rony.UnaryInputMeta(
		desc.WithField("category", desc.FieldMeta{
			Description: "Product category",
			Enum:        []string{"electronics", "books", "clothing"},
		}),
	),
	rony.UnaryHeader("Authorization", true), // required header
)
```

### Export docs to a file

```go
_ = srv.GenDocFile(context.Background(), "openapi.json")
```

---

## Generating Client Stubs

RonyKIT can generate type-safe client code in Go and TypeScript from your service
definitions.

### Go client stubs

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

### TypeScript client stubs

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

See `example/ex-04-stubgen` for a complete working example.

---

## File Uploads and Raw Bodies

### File uploads (multipart forms)

Use `kit.MultipartFormMessage` as your input type:

```go
func Upload(ctx *rony.SUnaryCtx, in kit.MultipartFormMessage) (*kit.RawMessage, error) {
	form := in.GetForm()
	for name, files := range form.File {
		// Process uploaded files
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

Use `kit.RawMessage` when you need to handle the body as raw bytes:

```go
func RawEcho(ctx *rony.SUnaryCtx, in kit.RawMessage) (*kit.RawMessage, error) {
	return &in, nil
}

rony.Setup(srv, "RawService", rony.EmptyState(),
	rony.WithUnary(RawEcho, rony.POST("/raw")),
)
```

### HTTP redirects

Access the REST connection to send redirects:

```go
func RedirectHandler(ctx *rony.SUnaryCtx, in RedirectRequest) (*rony.EMPTY, error) {
	if rc, ok := ctx.RESTConn(); ok {
		rc.Redirect(307, in.URL)
	}
	return nil, nil
}
```

---

## Testing

### Running the test suite

```bash
make setup    # install tools (gotestsum, golangci-lint) — run once
make test     # run tests across all modules
make lint     # lint all modules
```

To test a single module:

```bash
cd rony && go test ./...
```

### Testing your handlers

Since handlers are regular Go functions with typed inputs and outputs, you can
unit test them directly:

```go
func TestGreet(t *testing.T) {
	resp, err := Greet(testCtx, GreetRequest{Name: "World"})
	assert.NoError(t, err)
	assert.Equal(t, "Hello, World!", resp.Message)
}
```

For integration tests, use the generated Go client stubs to call your endpoints
end-to-end. For low-level `kit.Context` testing, see `kit/ctx_testkit.go`.

---

## Production Checklist

Before deploying to production, make sure you have:

- [ ] **Error handling** — consistent error codes from `rony/errs` across all handlers
- [ ] **Input validation** — DTOs properly tagged; optional fields use pointer types
- [ ] **API docs** — enriched with field metadata, enums, and descriptions
- [ ] **CORS** — configured with `rony.WithCORS(rony.CORSConfig{...})`
- [ ] **Compression** — enabled with `rony.WithCompression(...)`
- [ ] **Auth middleware** — in place for protected endpoints
- [ ] **Observability** — tracer and logger wired with `rony.WithTracer` / `rony.WithLogger`
- [ ] **Timeouts** — set at reverse proxy and service layer with `rony.WithShutdownTimeout`
- [ ] **Versioning** — stable URL paths (e.g., `/v1/...`); deprecated endpoints marked
- [ ] **Client stubs** — generated and distributed to consumers
- [ ] **Tests pass** — `make test` and `make lint` both green

---

## Useful Server Options

Here is a quick reference of common `rony.NewServer` options:

| Option                   | Description                                 |
|--------------------------|---------------------------------------------|
| `Listen(":8080")`        | Bind address for the HTTP gateway           |
| `WithServerName("name")` | Server name (appears in docs and logs)      |
| `WithVersion("v1.0.0")`  | API version                                 |
| `WithCORS(config)`       | Cross-origin resource sharing               |
| `WithCompression(level)` | Response compression (gzip/zstd)            |
| `WithAPIDocs("/docs")`   | Serve OpenAPI docs at the given path        |
| `UseSwaggerUI()`         | Use Swagger UI instead of the default ReDoc |
| `WithTracer(tracer)`     | Plug in a distributed tracer                |
| `WithLogger(logger)`     | Plug in a structured logger                 |
| `WithPrefork()`          | Multi-process mode for higher throughput    |
| `WithShutdownTimeout(d)` | Graceful shutdown timeout                   |
| `WithErrorHandler(fn)`   | Override global error serialization         |

---

## Context Helpers

The handler context (`rony.UnaryCtx` / `rony.SUnaryCtx`) provides helpers for
common tasks:

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

## Next Steps

- Browse the [runnable examples](./example) for real-world patterns.
- Read the [rony package docs](./rony/README.MD) for API details.
- Read the [kit package docs](./kit/README.MD) if you need low-level control.
- Check `example/ex-04-stubgen` for a full stub generation workflow.
