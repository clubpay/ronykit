---
import_path: github.com/clubpay/ronykit/x/telemetry/logkit
short_name: logkit
---

OpenTelemetry-bridged structured logger (zap-based) with OTLP/stdout export. The only logger in service code.

## Usage Hint

```go
logger := logkit.New() // or injected *logkit.Logger via fx
appLog := logger.With("APP")
appLog.Info("created item", "id", id)
```

- Construct with `logkit.New(opts ...Option) *Logger`.
- Inject `*logkit.Logger` via fx; create children with `.With(name)`.
- Never import `log`, `log/slog`, or `go.uber.org/zap` — denied by depguard.

Do not use stdlib logging "just for a script" inside a feature module.
