---
import_path: github.com/clubpay/ronykit/rony
short_name: rony
---

Batteries-included application framework on top of `kit`. Scaffolded feature handlers, contracts, and server wiring live here. Read this
before writing `api/` or `pkg/runner` code.

## Usage Hint

**`Desc()` vs `Setup`:** feature `api/service.go` returns options. The server applies them.

```go
type RContext = rony.UnaryCtx[rony.EMPTY, rony.NOP] // alias: rony.SUnaryCtx

func (svc Service) Desc() rony.SetupOption[rony.EMPTY, rony.NOP] {
	return rony.SetupOptionGroup[rony.EMPTY, rony.NOP](
		rony.WithUnary(svc.CreateItem, rony.POST("/v1/items")),
		rony.WithStream(svc.WatchItems, rony.RPC("items.Watch")),
	)
}

// pkg/runner — server wiring only
rony.Setup(srv, "items", rony.EmptyState(), itemsmod.App().Desc())
```

- Typed JSON API → `rony.WithUnary` / `WithRawUnary`. Handler:
  `func (svc Service) Name(ctx *RContext, in Input) (*Output, error)` — input is **not** a pointer.
- RPC / WebSocket stream → `rony.WithStream`.
- Per-request HTTP/WebSocket proxy after handler logic → `rony.WithRelay` + `RelayCtx.Relay()` (read `architecture/handler-relay`). Catch-all
  path: `rony.RelayALL`, not `rony.ANY` (it does not exist). Unary catch-all is `rony.ALL`.
- Static path → fixed upstream at gateway start → `rony.WithReverseProxy`.
- Docs on the server → `rony.WithAPIDocs(path)`. Standalone generator → `packages/apidoc`.
- Client stubs → `rony.GenerateStub` (read `packages/stub`).

Do **not** build `kit.Contract` / `desc.Service` by hand in scaffolded `api/` handlers. Do **not** call `rony.Setup` from `Desc()`.
For raw `kit` (custom gateways/clusters/codecs) read `packages/kit`.
