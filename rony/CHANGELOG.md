# Changelog

All notable changes to the `rony` module are documented here.

## Unreleased

### Fixed

- **`ReduceState`** now unlocks via `defer`, so a panic in `Reduce` or the callback cannot leave the service lock held.
- **`Server` lifecycle** — `Stop`, `PrintRoutes`, `PrintRoutesCompact`, and `LogEndpoints` no-op if `Start` was never called. `initEdge` no longer appends docs `WithServeFS` onto the long-lived gateway option list (Start/Stop/Start no longer accumulates duplicate mounts).
- **`errs`** — `ErrCode.String`, `ErrCode.HTTPStatus`, and `Error.GetCode` no longer panic on an out-of-range code (they report `unknown` / 500). `Wrap` now copies an `HTTPStatus` override the same way `Builder.Cause` already did. `WrapCode` still does not copy the override, so the new code's default status applies.

### Added

- **`RelayCtx`** and **`SRelayCtx`** — relay-only handler context (no envelope output helpers). Exposes `Relay()`, `InputBody()`, `RESTConn()`, `IsWebSocketUpgrade()`.
- **`WithRelay`** setup option and **`registerRelay`** registration path (`setup_relay.go`). Separate from `WithUnary` / `WithRawUnary`; success never auto-`Send()`s a JSON envelope.
- Route helpers: **`RelayALL`**, **`RelayGET`**, **`RelayPOST`**, etc., plus **`RelayMiddleware`**, **`RelayDecoder`**, **`RelayName`**, **`RelayDeprecated`**.

### Notes

- Relay routes **must** use `WithRelay` + `RelayCtx`. Do not use `WithUnary` for passthrough proxy endpoints — `UnaryCtx` has no relay API by design.
