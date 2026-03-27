Log exclusively through `x/telemetry/logkit` (OpenTelemetry-bridged zap).

- Inject `*logkit.Logger` via fx.
- Use `.With()` for contextual fields.
- Never import raw `zap`, `slog`, or `log`.
