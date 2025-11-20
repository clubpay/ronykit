### Goal

This guide teaches you how to build production‑grade servers using the `rony` package on top of RonyKIT. It distills
best practices from the code in `rony` and `kit`, and from the examples under `example/`.

- When to use `rony` vs `kit`
- Project scaffolding with `ronyup`
- Defining APIs and generating stubs/Swagger from descriptions
- Writing handlers with strong contexts (unary and stream)
- State management and middleware patterns
- Handling common server tasks: RPC, REST, uploads, redirects, raw bodies
- Error handling with `rony/errs`
- Testing, tracing/logging, performance, deployment

Where code comes from:

- rony core: `rony/*.go` (server, setup, context, middleware)
- kit core: `kit/*` (context, gateway abstractions)
- gateway (default): `std/gateways/fasthttp/*`
- examples: `example/ex-01-rpc`, `ex-02-rest`, `ex-06-counter-stream`, `ex-09-mw`, `ex-04-stubgen`

---

### 1) `rony` vs `kit`

- `kit` is the low‑level engine: routing, gateway, descriptors (`desc`), envelopes, etc.
- `rony` is the opinionated higher‑level layer: easier server setup, strongly‑typed handler signatures, state pattern,
	simple middleware, docs generation.
- Choose `rony` for most services. Drop down to `kit` if you need to wire your own gateway or deeply customize routing.

Key types (rony):

- `rony.Server` — wraps a `kit.EdgeServer` and optional Swagger/ReDoc docs serving.
- `rony.Setup` — registers services/contracts/handlers with type‑safe signatures.
- `rony.UnaryHandler` and `rony.StreamHandler` — your handler function shapes.
- `rony.UnaryCtx` / `rony.StreamCtx` — strong context with helpers over `kit.Context`.
- `rony.State[A]` — optional shared state with reducer pattern.
- `rony.EMPTY`/`rony.NOP` — utility state/action for stateless services.

---

### 2) Scaffolding a new project with `ronyup`

Install the CLI and generate a project skeleton based on templates in `ronyup/internal/skeleton/...`.

```bash
go install github.com/clubpay/ronykit/ronyup@latest
ronyup setup -d ./my-service -m github.com/you/myservice -p MyService
```

The scaffold uses Uber FX to wire an API module and gives you:

- Service wrapper (`service.gotmpl`), API module (`api/service.gotmpl`), app wiring (`internal/app/app.gotmpl`), gateway
	module (`gateway/*.gotmpl`).
- A clean place to add `Desc()` and handlers.

Tip: Keep the generated FX structure — it simplifies lifecycle and dependency wiring.

---

### 3) Bootstrapping a `rony` server

Create a server and register services/contracts using `rony.Setup`.

```go
package main

import (
	"context"

	"github.com/clubpay/ronykit/rony"
)

func main() {
	srv := rony.NewServer(
		rony.WithServerName("my-service"),
		rony.WithVersion("v1.0.0"),
		rony.Listen(":8080"),
		rony.WithCORS(rony.CORSConfig{AllowOrigin: "*"}),
		rony.WithCompression(rony.CompressionLevelDefault),
		// Serve API docs at /docs using Swagger UI (or use ReDoc by default)
		rony.WithAPIDocs("/docs"),
		rony.UseSwaggerUI(),
	)

	// Register a service with NO shared state (uses EMPTY/NOP internally)
	rony.Setup(srv, "EchoService", rony.EmptyState(),
		rony.WithUnary(Echo, rony.GET("/echo/:id")),
	)

	// Start & block until shutdown
	_ = srv.Run(context.Background())
}

// Simple stateless handler (see section 5 for details)
type EchoIn struct {
	ID int64 `json:"id"`
}
type EchoOut struct {
	ID int64 `json:"id"`
}

func Echo(ctx *rony.SUnaryCtx, in EchoIn) (*EchoOut, error) {
	return &EchoOut{ID: in.ID}, nil
}
```

Useful server options (see `rony/server_options.go`):

- `Listen(":8080")` — bind address
- `WithServerName/WithVersion` — ID/version surfacing into docs/logs
- `WithCORS`, `WithCompression` — cross‑origin and gzip/zstd levels
- `WithAPIDocs` + `UseSwaggerUI()` or default ReDoc — serve generated docs
- `WithTracer`, `WithLogger`, `WithPrefork`, `WithShutdownTimeout`
- `WithErrorHandler` — override error serialization if needed

---

### 4) Describing your API and generating stubs/Swagger

You get two complementary ways to describe contracts:

A) Describe in code as you register handlers with `rony.WithUnary/WithStream` using options:

- `GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS(path)` or generic `REST(method, path)`
- `RPC(predicate)` for streaming routes
- `UnaryInputMeta`/`UnaryOutputMeta` to add field metadata to DTOs
- `UnaryHeader` to declare required/optional headers
- `UnaryName`/`UnaryDeprecated` to control name and deprecation flags

B) Use `kit/desc` directly to build services/contracts (legacy or advanced control). `rony` can accept old
`desc.Contract` via `WithContract` during migrations.

Docs and stubs:

- Generate and serve OpenAPI: `srv.GenDoc`/`srv.GenDocFile` or `WithAPIDocs` + `UseSwaggerUI()` or default ReDoc UI.
- Generate client/server stubs (Go/TS) with `stub/stubgen` from service descriptions.

Recommended workflow:

```go
// After all rony.Setup(...) calls
// 1) Export service descriptions from the server
svcs := srv.ExportDesc()

// 2) Run stubgen to generate Go and/or TypeScript clients
import "github.com/clubpay/ronykit/stub/stubgen"

stubgen.New(
stubgen.WithGenEngine(stubgen.NewGolangEngine(stubgen.GolangConfig{PkgName: "mysvc"})),
stubgen.WithTags("json"),
stubgen.WithFolderName("mysvc"),
stubgen.WithStubName("myService"),
).MustGenerate(svcs...)

// Optionally generate TS client as well (see example/ex-04-stubgen)
```

Notes:

- Tag your DTO fields with the same tag your service encoding uses (default `json`). The REST decoder binds
	path/query/form/body values by that tag.
- `srv.ExportDesc()` makes it easy to reuse whatever you declared via `rony.Setup`.

---

### 5) Writing unary handlers with strong context

Handler signature:

```go
type UnaryHandler[S rony.State[A], A rony.Action, IN, OUT rony.Message] func (ctx *rony.UnaryCtx[S, A], in IN) (*OUT, error)
```

- Use `rony.SUnaryCtx` alias if you don’t need state: `func(ctx *rony.SUnaryCtx, in IN) (*OUT, error)`.
- Set up routes with `GET/POST/...` or `REST(method, path)`. Example from `example/ex-02-rest`:

```go
// Request is filled from path/query/body using json tags
type SumRequest struct {
Val1 int64 `json:"val1"` // can be bound from :val1 or query/body val1
Val2 int64 `json:"val2"`
}

type SumResponse struct { Val int64 `json:"val"` }

func Sum(ctx *rony.SUnaryCtx, in SumRequest) (*SumResponse, error) {
return &SumResponse{Val: in.Val1 + in.Val2}, nil
}

// Registration
rony.Setup(srv, "Calc", rony.EmptyState(),
rony.WithUnary(Sum,
rony.GET("/sum/:val1/:val2"),
rony.POST("/sum"),
),
)
```

Context helpers (see `rony/context.go`):

- Headers: `ctx.GetInHdr("X-Req-Id")`, `ctx.SetOutHdr("X-Resp","v")`
- Connection and user context: `ctx.Conn()`, `ctx.Context()`, `ctx.SetUserContext(...)`
- Short‑circuit next middleware: `ctx.StopExecution()`
- Access underlying `kit.Context`: `ctx.KitCtx()`
- REST only: `rc, ok := ctx.RESTConn()` to use `GetMethod/GetHost/GetPath/Redirect/SetStatusCode`.

Raw input/output:

- If you need raw bodies: make `IN` or `OUT` be `kit.RawMessage`.
- For file uploads (multipart): make `IN` be `kit.MultipartFormMessage` and then `frm := in.GetForm()`.

Example: file upload (see `example/ex-02-rest/api/service.go:152`):

```go
func Upload(ctx *rony.SUnaryCtx, in kit.MultipartFormMessage) (*kit.RawMessage, error) {
frm := in.GetForm()
// read frm.File / frm.Value as needed
empty := kit.RawMessage{}
return &empty, nil
}

rony.Setup(srv, "Files", rony.EmptyState(),
rony.WithUnary(Upload, rony.POST("/upload")),
)
```

Redirects (REST only):

```go
func RedirectTo(ctx *rony.SUnaryCtx, in struct{ URL string `json:"url"` }) (*rony.EMPTY, error) {
if rc, ok := ctx.RESTConn(); ok {
rc.Redirect(307, in.URL)
}
return nil, nil
}
```

Deprecation and naming:

- `UnaryDeprecated(true)` marks a route deprecated in docs.
- `UnaryName("MyRoute")` overrides the auto handler‑name.

---

### 6) State management with reducer pattern

`rony` lets you share mutable state across handlers.

```go
type MyState struct { sync.Mutex; Hits int }
func (s *MyState) Name() string { return "MyState" }
func (s *MyState) Reduce(action string) error {
switch action { case "hit": s.Hits++ }
return nil
}

func Hello(ctx *rony.UnaryCtx[*MyState, string], in struct{}) (*struct{}, error) {
_ = ctx.ReduceState("hit", nil) // atomically increments under lock if state implements sync.Locker
// read current state safely inside ReduceState callback if you need both mutation & read:
// _ = ctx.ReduceState("hit", func(s *MyState, err error) error { fmt.Println(s.Hits); return err })
return &struct{}{}, nil
}

rony.Setup(srv, "Greeter", rony.ToInitiateState(&MyState{}), rony.WithUnary(Hello, rony.GET("/hello")))
```

Notes:

- If your state type implements `sync.Locker`, `ReduceState` will lock around the mutation and callback.
- Prefer keeping state small and delegating heavy shared resources (DB pools, caches) to DI containers (FX) or global
	factories — expose them via the handler’s closure or module wiring.
- Use `rony.EmptyState()` when you don’t need state.

---

### 7) Middleware (stateless vs stateful)

- Stateless: simple `kit.HandlerFunc` executed before your handler. Great for auth, logging, request shaping.
- Stateful: gets `*rony.BaseCtx[S,A]` so it can access state.

Register via `rony.WithMiddleware(...)` during `rony.Setup`.

Stateless example:

```go
func Authz(next kit.HandlerFunc) kit.HandlerFunc {
return func (kc *kit.Context) {
if kc.In().GetHdr("Authorization") == "" { kc.Error(errs.B().Code(errs.PermissionDenied).Msg("missing auth").Err()); return }
next(kc)
}
}

rony.Setup(srv, "Secured", rony.EmptyState(),
rony.WithMiddleware[rony.EMPTY, rony.NOP](Authz),
rony.WithUnary(Sensitive, rony.GET("/secret")),
)
```

Stateful example:

```go
type BumpMW[S rony.State[A], A rony.Action] func (ctx *rony.BaseCtx[S, A])

func Bump[S rony.State[A], A rony.Action](ctx *rony.BaseCtx[S, A]) { _ = ctx.ReduceState(any("hit").(A), nil) }

rony.Setup(srv, "Stateful", rony.ToInitiateState(&MyState{}),
rony.WithMiddleware[ *MyState, string ](Bump[*MyState, string]),
rony.WithUnary(Hello, rony.GET("/hello")),
)
```

Tip: Prefer stateless middleware unless you truly need shared state access.

---

### 8) Streaming RPC handlers

Use when you need server push / bidirectional patterns. Register with `rony.WithStream` and `RPC(predicate)`.

```go
// in/out are typed messages
type ChatIn struct { Text string `json:"text"` }
type ChatOut struct { Text string `json:"text"` }

func Chat(ctx *rony.StreamCtx[*MyState, string, ChatOut], in ChatIn) error {
// push to this connection
ctx.Push(ChatOut{Text: "ack: " + in.Text})
return nil
}

rony.Setup(srv, "Chat", rony.ToInitiateState(&MyState{}),
rony.WithStream(Chat, rony.RPC("chatMessage")),
)
```

- Use `ctx.Push(out, rony.WithHdr(key,val))` to send envelopes; returns current state for chaining if desired.
- Behind the scenes, `fasthttp` gateway handles the streaming channel (see `std/gateways/fasthttp`).

---

### 9) Error handling with `rony/errs`

Handlers return `(*OUT, error)`. `rony` converts errors via `errs.Convert` and sets a default error shape on contracts.

Use the builder for structured errors:

```go
if invalid {
return nil, errs.B().Code(errs.InvalidArgument).Msg("bad input").Err()
}
```

Or wrap underlying errors with code:

```go
if err != nil { return nil, errs.WrapCode(err, errs.Internal, "db failed") }
```

Customize error serialization globally using:

```go
rony.NewServer(rony.WithErrorHandler(func (kc *kit.Context, err error) { /* write your envelope */ }))
```

Mapping to HTTP status codes is handled in the gateway; use codes from `errs` consistently so clients (including stubs)
can reason about them.

---

### 10) Binding rules (REST): path, query, body

The REST decoder (`std/gateways/fasthttp/decoder.go`):

- Collects path params (`:id`), query params, and body form fields.
- Binds into your `IN` struct by field tag of the service encoding (`json` by default). Example:

```go
type In struct {
ID   int64  `json:"id"`   // binds from /:id or ?id= or body {"id":...}
Name string `json:"name"` // binds from query/body name
// for optional fields, use pointers
Page *int32 `json:"page"` // only set when provided
}
```

Tips:

- Keep tag names simple and consistent (lower snake/camel) — they’re used across path/query/body.
- For headers, read them from context: `ctx.GetInHdr("X-Request-Id")`.
- For raw bodies use `kit.RawMessage`.
- For multipart bodies use `kit.MultipartFormMessage`.

---

### 11) Common recipes

A) Raw echo endpoint

```go
func RawEcho(ctx *rony.SUnaryCtx, in kit.RawMessage) (*kit.RawMessage, error) { return &in, nil }
rony.Setup(srv, "Raw", rony.EmptyState(), rony.WithUnary(RawEcho, rony.POST("/raw_echo")))
```

B) Redirect based on method

```go
func SumRedirect(ctx *rony.SUnaryCtx, in SumRequest) (*rony.EMPTY, error) {
if rc, ok := ctx.RESTConn(); ok {
switch rc.GetMethod() {
case "GET": rc.Redirect(307, fmt.Sprintf("http://%s/sum/%d/%d", rc.GetHost(), in.Val1, in.Val2))
case "POST": rc.Redirect(307, fmt.Sprintf("http://%s/sum", rc.GetHost()))
}
}
return nil, nil
}
```

C) Set response headers

```go
func WithHeaders(ctx *rony.SUnaryCtx, in In) (*Out, error) {
ctx.SetOutHdr("X-Trace-Id", ctx.GetInHdr("X-Trace-Id"))
return &Out{}, nil
}
```

D) Multiple routes to one handler

```go
rony.WithUnary(Echo,
rony.GET("/echo/:id", rony.UnaryName("EchoGET")),
rony.POST("/echo-post", rony.UnaryName("EchoPOST")),
)
```

E) Mark endpoint deprecated in docs

```go
rony.WithUnary(OldHandler, rony.GET("/old", rony.UnaryDeprecated(true)))
```

---

### 12) Generating and serving API docs

- Serve at runtime: `WithAPIDocs("/docs")` + optionally `UseSwaggerUI()`; otherwise, default ReDoc UI.
- Offline: `srv.GenDocFile(ctx, "openapi.json")`.
- You can also include header requirements and field metadata to improve docs: `UnaryHeader`,
	`UnaryInputMeta(desc.WithField(...))`.

Example (ex-02-rest shows field metadata usage):

```go
rony.WithUnary(Echo,
rony.GET("/echo/:id"),
rony.UnaryInputMeta(
desc.WithField("OptionalStrField", desc.FieldMeta{Enum: []string{"a", "b", "c"}}),
),
)
```

---

### 13) Client and server stub generation

Use `stub/stubgen` (see `example/ex-04-stubgen/stub/gen.go`). You can generate from any `[]desc.ServiceDesc` — export
from your server or define by hand.

```go
svcs := srv.ExportDesc()
stubgen.New(
stubgen.WithGenEngine(stubgen.NewGolangEngine(stubgen.GolangConfig{PkgName: "mysvc"})),
stubgen.WithTags("json"),
stubgen.WithFolderName("mysvc"),
stubgen.WithStubName("myService"),
).MustGenerate(svcs...)
```

For TypeScript clients:

```go
stubgen.New(
stubgen.WithGenEngine(stubgen.NewTypescriptEngine(stubgen.TypescriptConfig{GenerateSWR: true})),
stubgen.WithTags("json"),
stubgen.WithFolderName("mysvc-ts"),
stubgen.WithStubName("myService"),
).MustGenerate(svcs...)
```

Generated Go stubs also include mock helpers for testing (`StubMock` etc.).

---

### 14) Testing strategy

- Prefer the project’s curated test runner: `make test` (uses `scripts/run-test.sh`).
- Unit testing handlers: call them directly with typed inputs; mock out state or DI.
- Integration testing via gateway: use generated Go stubs to call endpoints end‑to‑end.
- Low‑level context tests: `kit/ctx_testkit.go` provides a test connection/context if you need to test `kit.Context`
	level code.
- Test environments: `util/testkit` provides helpers and wiremock‑style fixtures; use when you integrate external
	systems.

---

### 15) Security, observability, performance

- CORS: configure early with `WithCORS`.
- Tracing/logging: `WithTracer` and `WithLogger` (plug your tracer/logger implementation from `kit`). Also see
	`contrib/tracekit` for helpers.
- Rate limiting and quotas: compose stateless middleware; `util/ratelimit` can help implement token buckets.
- Compression: choose `WithCompression(...)` to balance performance vs CPU.
- Prefork: `WithPrefork()` for multi‑process mode where appropriate.
- Error hygiene: always return `errs` codes; wrap internal errors with `errs.WrapCode`.
- Validation: use DTO types to model required/optional fields (pointer = optional). Surface enums, min/max in docs via
	`UnaryInputMeta(desc.WithField(...))`.
- Versioning: bake `WithVersion` and use `/v1`-style paths for breaking changes.

---

### 16) Deployment, lifecycle, and routes

- Start/Stop: `srv.Start(ctx)` and `srv.Stop(ctx, os.Interrupt)` if you manage lifecycle manually (FX hooks, etc.).
- Blocking run: `srv.Run(ctx)` starts, prints route table, and waits for stop signals.
- Inspect routes: `srv.PrintRoutes(os.Stdout)` or `PrintRoutesCompact` during boot.

---

### 17) Suggested conventions and pitfalls

- DTO tags: align with service encoding (default `json`). This controls binding from path/query/body.
- Keep handlers pure: compute, return `(*OUT, error)`. Avoid writing directly to `kit.Context` unless doing raw or
	advanced use.
- Use `rony.WithRawUnary` only when you truly need to control raw envelopes or custom serialization.
- For file uploads, set server limits in your gateway environment (reverse proxy or fasthttp tunables) and validate
	filenames/sizes.
- Don’t share large mutable state via `rony.State`; use DI (FX) to provide DB/cache clients.
- Use `UnaryDeprecated(true)` before removing endpoints; keep docs honest.

---

### 18) Minimal end‑to‑end example

```go
// main.go
package main

import (
	"context"

	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/rony"
)

type In struct {
	Name string `json:"name"`
}
type Out struct {
	Hello string `json:"hello"`
}

func Hello(ctx *rony.SUnaryCtx, in In) (*Out, error) { return &Out{Hello: "Hi, " + in.Name}, nil }

func main() {
	srv := rony.NewServer(
		rony.WithServerName("hello"), rony.WithVersion("v1"), rony.Listen(":8080"),
		rony.WithAPIDocs("/docs"), rony.UseSwaggerUI(),
	)

	rony.Setup(srv, "Greeter", rony.EmptyState(),
		rony.WithUnary(Hello,
			rony.GET("/hello/:name"),
			// Improve docs with metadata
			rony.UnaryInputMeta(desc.WithField("name", desc.FieldMeta{Description: "Person's name"})),
		),
	)

	_ = srv.Run(context.Background())
}
```

Open http://localhost:8080/docs for live API docs.

---

### 19) Checklist for production readiness

- [ ] Error taxonomy agreed and consistently returned (`rony/errs`)
- [ ] DTOs tagged and validated; docs enriched with metadata
- [ ] CORS/compression configured; timeouts at reverse proxy and service layer
- [ ] AuthN/Z middleware in place
- [ ] Observability wired (tracer/logger); route table printed on boot
- [ ] File upload size/content‑type validation
- [ ] Stable versioning strategy; deprecations marked
- [ ] Stubs generated and integrated into clients
- [ ] `make test` passes; examples build where applicable

---

### 20) Targeted questions for you

To tailor the guide and templates to your preferences, please confirm:

1) Docs UI default: prefer Swagger UI or ReDoc by default for `WithAPIDocs`?
2) Error payload conventions: do you want a custom error envelope (e.g., map `errs.ErrCode` to specific HTTP codes), or
	 keep the default?
3) Upload limits: what maximum file size and accepted content types should we enforce at the gateway level?
4) Naming/style: do you prefer kebab or snake in URLs and lowerCamel in JSON tags? Any strict rules for route names (
	 `UnaryName`) and tag sets for `stubgen`?
5) Security defaults: should the scaffold include a default auth middleware and rate limiter example?

If you share your preferences, I’ll refine this guide with concrete defaults and optionally adjust the `ronyup`
templates to match them.
