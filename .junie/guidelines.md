# Junie Project Guidelines — RonyKIT

These guidelines tell Junie how to work effectively in this repository.

Project overview
- RonyKIT is a collection of Go modules that together provide an extendable and high‑performance toolkit for building API/Edge servers.
- The framework is split into multiple modules and standard implementations, and organized as a Go workspace.

Repository layout (high level)
- Core modules: kit, rony, flow, stub, boxship, ronyup, contrib, util
- Standard implementations:
  - Gateways: std/gateways/fasthttp, std/gateways/fastws, std/gateways/silverhttp
  - Clusters: std/clusters/p2pcluster, std/clusters/rediscluster
- Examples: example/ex-01-rpc, ex-02-rest, ex-03-cluster, ex-04-stubgen, ex-05-counter, ex-06-counter-stream, ex-08-echo, ex-09-mw
- Docs and assets: docs/

Daily workflow for Junie
1) Install tools (once per environment)
   - make setup
   - This installs gotestsum which the test script relies on.

2) Run tests (required before submitting changes)
   - Preferred: make test
   - What it does: runs scripts/run-test.sh, which iterates over key modules and examples, executes go test with coverage via gotestsum, and summarizes coverage with go tool cover.
   - Note: Because this is a multi-module workspace, running go test ./... at the repo root will not mirror the curated list used by the project; use make test.

3) Build/compile
   - There is no single top-level build target; build individual modules or examples with standard Go commands:
     - Example: cd example/ex-01-rpc && go build ./...
   - Some modules contain tools/CLIs (e.g., ronyup, boxship) that can be built similarly.

4) Formatting and basic checks
   - Run go fmt ./... and go vet ./... in the module you modified (or at repo root for a quick pass) before submitting.
   - Keep imports organized and run go mod tidy in any module where dependencies changed.

5) Submitting changes checklist
   - Keep edits minimal and targeted to the issue at hand.
   - Update or add documentation when behavior changes (README.MD or relevant package README).
   - Run make test and ensure all tests in the curated list pass.
   - Ensure the repository still builds for affected modules.

Makefile targets
- setup: installs required tools (gotestsum)
- test: runs the project’s curated test suite via scripts/run-test.sh
- cleanup: executes scripts/cleanup.sh for housekeeping
- new-version-patch / new-version-minor: internal version bump helpers

Useful references
- Project README: README.MD (overview, structure, and basic commands)
- Contribution guide: CONTRIBUTING.md
- Licenses and policies: LICENSE, CODE_OF_CONDUCT.md, COMPLIANCE.md

Notes for non-standard behavior
- Tests: The repository uses a curated list of packages and examples for testing (see scripts/run-test.sh). Always prefer make test over ad‑hoc go test ./... at the workspace root.
- Coverage: scripts/run-test.sh generates coverage.out per package and summarizes with go tool cover.


## Rony Package Guidelines — For Junie

This section is a comprehensive, action‑oriented guide for Junie to build services using the `rony` package (the higher‑level layer on top of `kit`). It consolidates patterns from the `rony` and `kit` packages and from examples under `example/`.

When to choose rony vs kit
- Use `rony` for most API services: simpler server setup, typed handlers, reducer‑style shared state, simple middleware, built‑in docs.
- Drop down to `kit` only when you need custom gateways, very fine routing control, or experimental wiring.

Quick reference to source code
- rony core: rony/ (server, setup, context, middleware)
- kit core: kit/ (context, descriptors, gateways abstractions)
- default gateway: std/gateways/fasthttp/
- examples: example/ex-01-rpc, ex-02-rest, ex-06-counter-stream, ex-09-mw, ex-04-stubgen


### 1) Scaffolding a new project (ronyup)
Install and use the CLI to bootstrap a service skeleton (Uber FX wiring + API module):

```bash
go install github.com/clubpay/ronykit/ronyup@latest
ronyup setup -d ./my-service -m github.com/you/myservice -p MyService
```

Keep the generated FX structure. Implement your API in the `api` module and expose a `Desc()` that returns `rony.SetupOption[...]` for registration.


### 2) Bootstrapping a rony server
Create a server and register handlers via `rony.Setup`:

```go
srv := rony.NewServer(
  rony.WithServerName("my-service"),
  rony.WithVersion("v1.0.0"),
  rony.Listen(":8080"),
  rony.WithCORS(rony.CORSConfig{AllowOrigin: "*"}),
  rony.WithCompression(rony.CompressionLevelDefault),
  rony.WithAPIDocs("/docs"), // serve API docs
  rony.UseSwaggerUI(),        // or keep default ReDoc
)

// Stateless service (EMPTY/NOP state)
rony.Setup(srv, "EchoService", rony.EmptyState(),
  rony.WithUnary(Echo, rony.GET("/echo/:id")),
)

_ = srv.Run(context.Background())
```

Useful server options (rony/server_options.go)
- Listen, WithServerName, WithVersion
- WithCORS, WithCompression, WithPrefork, WithShutdownTimeout
- WithAPIDocs + UseSwaggerUI (or default ReDoc)
- WithTracer, WithLogger, WithErrorHandler


### 3) Defining APIs and generating docs/stubs
Describe REST routes and RPC streams as you register handlers. For unary routes:
- Use `GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS(path)` or `REST(method, path)`
- Enrich docs with `UnaryInputMeta`, `UnaryOutputMeta`, and `UnaryHeader`
- Control route naming with `UnaryName` and deprecations with `UnaryDeprecated`

Docs:
- Serve at runtime: `WithAPIDocs("/docs")` (+ `UseSwaggerUI()` if desired)
- Offline generation: `srv.GenDocFile(ctx, "openapi.json")`

Stubs (Go/TS):
- Export your service descriptions: `svcs := srv.ExportDesc()`
- Use `stub/stubgen` to generate:

```go
stubgen.New(
  stubgen.WithGenEngine(stubgen.NewGolangEngine(stubgen.GolangConfig{PkgName: "mysvc"})),
  stubgen.WithTags("json"),
  stubgen.WithFolderName("mysvc"),
  stubgen.WithStubName("myService"),
).MustGenerate(svcs...)
```

TypeScript client:
```go
stubgen.New(
  stubgen.WithGenEngine(stubgen.NewTypescriptEngine(stubgen.TypescriptConfig{GenerateSWR: true})),
  stubgen.WithTags("json"),
  stubgen.WithFolderName("mysvc-ts"),
  stubgen.WithStubName("myService"),
).MustGenerate(svcs...)
```


### 4) Handlers and strong context (unary)
Handler signature:
```go
type UnaryHandler[S rony.State[A], A rony.Action, IN, OUT rony.Message] func(ctx *rony.UnaryCtx[S,A], in IN) (*OUT, error)
```
- Use alias `rony.SUnaryCtx` when you don’t need shared state.
- Binding: the REST decoder fills `IN` from path/query/body using the service encoding tag (`json` by default). Use pointers for optional fields.
- Headers: `ctx.GetInHdr(key)`, `ctx.SetOutHdr(key, val)`
- REST connection: `rc, ok := ctx.RESTConn()` for `GetMethod/GetHost/GetPath/Redirect/SetStatusCode`.
- Raw bodies: use `kit.RawMessage` as `IN` or `OUT`.
- File uploads: use `kit.MultipartFormMessage` as `IN` and read `in.GetForm()`.

Examples (see `example/ex-02-rest/api/service.go`):
```go
type SumRequest struct { Val1 int64 `json:"val1"`; Val2 int64 `json:"val2"` }
type SumResponse struct { Val int64 `json:"val"` }
func Sum(ctx *rony.SUnaryCtx, in SumRequest) (*SumResponse, error) { return &SumResponse{Val: in.Val1+in.Val2}, nil }

func Upload(ctx *rony.SUnaryCtx, in kit.MultipartFormMessage) (*kit.RawMessage, error) {
  _ = in.GetForm() // validate files/fields
  empty := kit.RawMessage{}
  return &empty, nil
}
```

Deprecation and naming for docs:
```go
rony.WithUnary(Old, rony.GET("/old", rony.UnaryDeprecated(true)))
rony.WithUnary(Echo, rony.GET("/echo/:id", rony.UnaryName("EchoGET")))
```


### 5) Shared state with reducer pattern
Define a state and reduce actions atomically when your state implements `sync.Locker`.

```go
type MyState struct { sync.Mutex; Hits int }
func (s *MyState) Name() string { return "MyState" }
func (s *MyState) Reduce(action string) error { if action=="hit" { s.Hits++ }; return nil }

func Hello(ctx *rony.UnaryCtx[*MyState,string], in struct{}) (*struct{}, error) {
  _ = ctx.ReduceState("hit", nil)
  return &struct{}{}, nil
}

rony.Setup(srv, "Greeter", rony.ToInitiateState(&MyState{}), rony.WithUnary(Hello, rony.GET("/hello")))
```

Guidance:
- Keep shared state small; use DI (FX) for DB/cache clients.
- Use `rony.EmptyState()` if no state is needed.


### 6) Middleware
- Stateless middleware: plain `kit.HandlerFunc` (ideal for auth/logging/shaping)
- Stateful middleware: `func(ctx *rony.BaseCtx[S,A])` to access state
Register via `rony.WithMiddleware(...)` during `rony.Setup`.

Example (stateless auth):
```go
func Authz(next kit.HandlerFunc) kit.HandlerFunc {
  return func(kc *kit.Context) {
    if kc.In().GetHdr("Authorization")=="" { kc.Error(errs.B().Code(errs.PermissionDenied).Msg("missing auth").Err()); return }
    next(kc)
  }
}
```


### 7) Streaming RPC handlers
Register with `rony.WithStream` and `RPC(predicate)`; push messages with `ctx.Push`.

```go
type ChatIn struct { Text string `json:"text"` }
type ChatOut struct { Text string `json:"text"` }
func Chat(ctx *rony.StreamCtx[*MyState,string,ChatOut], in ChatIn) error { ctx.Push(ChatOut{Text: "ack: "+in.Text}); return nil }
rony.Setup(srv, "Chat", rony.ToInitiateState(&MyState{}), rony.WithStream(Chat, rony.RPC("chatMessage")))
```


### 8) Errors with `rony/errs`
Return structured errors from handlers:
```go
if invalid { return nil, errs.B().Code(errs.InvalidArgument).Msg("bad input").Err() }
if err != nil { return nil, errs.WrapCode(err, errs.Internal, "db failed") }
```
Customize globally via `WithErrorHandler` if needed.


### 9) Binding rules (REST)
- Path params `:id`, query params, and form fields bind into `IN` by the encoding tag (`json` default).
- Use pointer fields for optional values.
- Headers aren’t auto‑bound; read via context.

```go
type In struct { ID int64 `json:"id"`; Name string `json:"name"`; Page *int32 `json:"page"` }
```


### 10) Docs and stubs workflow
- Serve live docs: `WithAPIDocs("/docs")` (+ `UseSwaggerUI()` or default ReDoc)
- Offline docs: `srv.GenDocFile(ctx, "openapi.json")`
- Stubs: `svcs := srv.ExportDesc()` then `stubgen.New(...).MustGenerate(svcs...)`


### 11) Testing, security, observability, performance
- Testing: prefer `make test` (curated runner). Unit‑test handlers directly; use generated Go stubs for integration tests.
- Security: implement auth/ratelimiting as stateless middleware; validate upload sizes/types.
- Observability: add `WithTracer`, `WithLogger` (see `contrib/tracekit`, `util/log`).
- Performance: tune compression with `WithCompression`; consider `WithPrefork` where appropriate.
- Versioning: set `WithVersion` and use `/v1` paths; mark old endpoints with `UnaryDeprecated(true)` before removal.


### 12) Minimal end‑to‑end example
```go
type In struct{ Name string `json:"name"` }
type Out struct{ Hello string `json:"hello"` }
func Hello(ctx *rony.SUnaryCtx, in In) (*Out, error) { return &Out{Hello: "Hi, "+in.Name}, nil }

srv := rony.NewServer(rony.WithServerName("hello"), rony.WithVersion("v1"), rony.Listen(":8080"), rony.WithAPIDocs("/docs"), rony.UseSwaggerUI())
rony.Setup(srv, "Greeter", rony.EmptyState(), rony.WithUnary(Hello, rony.GET("/hello/:name")))
_ = srv.Run(context.Background())
```


### 13) Production checklist (for PRs)
- [ ] Errors use `rony/errs` with meaningful codes
- [ ] DTOs are properly tagged; docs enriched with metadata where helpful
- [ ] CORS/compression configured; timeouts set appropriately
- [ ] Auth/ratelimit middleware in place where applicable
- [ ] Tracing/logging configured; routes printed on boot
- [ ] Upload size/content‑type validated if used
- [ ] Versioning documented; deprecated endpoints marked
- [ ] Stubs generated and delivered to clients if needed
- [ ] `make test` passes; affected modules build
