---
keywords:
- relay
- reverse proxy
- websocket
- passthrough
- deploy
- session
applies_to_files:
- api
---

Handler-initiated HTTP/WebSocket relay forwards an inbound request to a **dynamic upstream URL** after your business logic runs (auth, session lookup, target resolution). Use this for session-scoped proxies (e.g. nested-agents deploy session relay), not for static gateway proxies.

## When to use what

| Need | Use |
|------|-----|
| Static path → fixed upstream at gateway startup | `rony.WithReverseProxy(path, proxy.WithAddress(...))` |
| Per-request upstream after handler logic | `rony.WithRelay` + `RelayCtx.Relay()` |
| Typed JSON API with envelope I/O | `rony.WithUnary` / `WithRawUnary` |

Relay is **mechanism only** — auth, target URL, path rewrite, and query stripping are **application policy** in the handler.

## Rony registration (required path)

Relay routes use a **separate setup path**. Do **not** use `WithUnary`, `WithRawUnary`, or `kit.RawMessage` unary hacks for proxy endpoints.

```go
rony.WithRelay(
    func(ctx *rony.SRelayCtx) error {
        target, err := svc.resolveRelayTarget(ctx) // auth + port-forward / upstream URL
        if err != nil {
            return err
        }
        return ctx.Relay(target, kit.RelayConfig{
            DropQueryParams: []string{"token"},
        })
    },
    rony.ANY("/deploy/v1/sessions/:sessionId/api/{path:*}"),
)
```

- Handler signature: `func(ctx *RelayCtx[S, A]) error` (alias `SRelayCtx` when state is unused).
- **`RelayCtx`** exposes `Relay`, `InputBody`, `RESTConn`, `IsWebSocketUpgrade`, plus shared `BaseCtx` helpers (`GetInHdr`, `Context()`, `Set`/`Get`, etc.).
- **`UnaryCtx`** and **`StreamCtx`** do **not** expose `Relay()` — compile-time separation.
- On success, the gateway writes the upstream HTTP/WS response **raw** (no envelope JSON). On error, return `error` — the `WithRelay` wrapper sends the standard `rony/errs` envelope.
- Service-level middleware (`WithMiddleware`) still runs; shared auth middleware can use `*kit.Context` as today.

### Route helpers

Same REST shape as unary routes: `RelayALL`, `RelayGET`, `RelayPOST`, `RelayPUT`, `RelayDELETE`, `RelayPATCH`, `RelayHEAD`, `RelayOPTIONS`, `RelayMiddleware`, `RelayDecoder`, `RelayName`, `RelayDeprecated`. `WithBasePath` applies to subsequent `WithRelay` routes.

## Kit layer (gateway authors / advanced)

- **`kit.RelayConn`** — optional interface on REST connections (`RelayHTTP`, `RelayWebSocket`, `RequestBody`, `WriteHTTPResponse`, …).
- **`kit.Relay(ctx, targetURL, cfg)`** — auto-selects HTTP vs WebSocket from the `Upgrade` header.
- **`kit.RelayHTTP` / `kit.RelayWebSocket`** — call `StopExecution()` on success (`ErrRelayCompleted`).
- Implemented on `std/gateways/fasthttp` `*httpConn` via `proxy.RelayHTTP` / `proxy.RelayWebSocket` (reuses hop-header logic from the static reverse proxy package).

### RelayConfig highlights

```go
kit.RelayConfig{
    ExtraRequestHeaders: map[string]string{"X-Custom": "1"},
    DropRequestHeaders:  []string{"Authorization"}, // plus default hop-by-hop headers
    DropQueryParams:     []string{"token"},
    TLSConfig:           tlsConfig,
    Timeout:             30 * time.Second,
    RewriteRequest:      func(req *kit.RelayRequestView) error { ... },
    RewriteResponse:     func(resp *kit.RelayResponseView) error { ... },
    WebSocketSubprotocols: []string{"binary", "base64"},
    WebSocketCheckOrigin:  func(origin string) bool { return true },
}
```

- `targetURL` must be **absolute** (`http://`, `https://`, `ws://`, `wss://`).
- Inbound **HTTP method** is preserved; path/query/host come from `targetURL` (build the URL in the handler, e.g. from route params + port-forward host).
- `RewriteResponse` buffers the full upstream body (v1); streaming rewrite is out of scope.

## Testing

- Kit: `kit/relay_test.go`, `TestContext.RunWithConn` for custom `RelayConn` fakes.
- Fasthttp: `std/gateways/fasthttp/conn_relay_test.go`.
- Rony: `rony/setup_relay_test.go`.

## Related docs

- `std/gateways/fasthttp/proxy/README.md` — static vs handler relay
- `kit/CHANGELOG.md`, `rony/CHANGELOG.md`
- Monorepo `docs/cookbook.md` — Handler relay section
