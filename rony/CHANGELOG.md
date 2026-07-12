# Changelog

All notable changes to the `rony` module are documented here.

## Unreleased

### Added

- **`RelayCtx`** and **`SRelayCtx`** — relay-only handler context (no envelope output helpers). Exposes `Relay()`, `InputBody()`, `RESTConn()`, `IsWebSocketUpgrade()`.
- **`WithRelay`** setup option and **`registerRelay`** registration path (`setup_relay.go`). Separate from `WithUnary` / `WithRawUnary`; success never auto-`Send()`s a JSON envelope.
- Route helpers: **`RelayALL`**, **`RelayGET`**, **`RelayPOST`**, etc., plus **`RelayMiddleware`**, **`RelayDecoder`**, **`RelayName`**, **`RelayDeprecated`**.

### Notes

- Relay routes **must** use `WithRelay` + `RelayCtx`. Do not use `WithUnary` for passthrough proxy endpoints — `UnaryCtx` has no relay API by design.
