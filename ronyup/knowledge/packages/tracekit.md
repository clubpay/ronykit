---
import_path: github.com/clubpay/ronykit/x/telemetry/tracekit
short_name: tracekit
---
Distributed tracing with W3C/B3 propagation integrated with kit.Tracer.

## Usage Hint

Use tracekit.W3C() or tracekit.B3() as the gateway Tracer; wrap cross-service calls with tracekit.NewSpan.
