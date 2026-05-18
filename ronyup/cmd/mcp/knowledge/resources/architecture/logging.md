Log exclusively through `x/telemetry/logkit` (OpenTelemetry-bridged zap).

- Inject `*logkit.Logger` via fx.
- Use `.With(name)` to attach a sub-logger name segment (for example
  `logger.With("APP")` or `.With("REPO")`); each call returns a new `*Logger`.
- Pass structured fields per call using the `logkit.Field` variadic of
  `Debug`/`Info`/`Warn`/`Error` (or the `*Ctx` variants for context-aware logs).
- Never import raw `zap`, `slog`, or `log`.
