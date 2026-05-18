---
import_path: github.com/clubpay/ronykit/x/telemetry/logkit
short_name: logkit
---
OpenTelemetry-bridged structured logger (zap-based) with OTLP/stdout export.

## Usage Hint

Inject *logkit.Logger via fx; use .With() for context; never import raw zap/slog/log.
