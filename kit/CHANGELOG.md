# Changelog

All notable changes to the `kit` module are documented here.

## v0.27.0

### Added

- **`RelayConn`** optional connection interface (`kit/conn.go`) for handler-initiated HTTP and WebSocket reverse proxying on gateways that implement it (e.g. `std/gateways/fasthttp`).
- **`kit.RelayConfig`**, **`kit.RelayHTTP`**, **`kit.RelayWebSocket`**, **`kit.Relay`** helpers (`kit/relay.go`). Successful relay returns `ErrRelayCompleted` internally and calls `StopExecution()` so the handler chain does not send an envelope response.
- **`Context.InputBody()`** — inbound body via `RelayConn.RequestBody()` when available, otherwise `InputRawData()`.
- **`Context.RelayConn()`** / **`Context.IsRelay()`** type assertions.

### Notes

- Handler relay is **dynamic** (per-request target URL decided in the handler). For **static** gateway-level proxying, continue using `rony.WithReverseProxy` / `fasthttp.WithReverseProxy`.
