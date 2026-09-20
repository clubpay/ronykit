---
import_path: github.com/clubpay/ronykit/x/telemetry/tracekit
short_name: tracekit
---

Distributed tracing with W3C/B3 propagation, used as the gateway `kit.Tracer`. Prefer this over raw OpenTelemetry tracer setup.

## Usage Hint

```go
tracer := tracekit.B3("mysvc") // or tracekit.W3C("mysvc")
ctx, span, end := tracekit.NewSpan(ctx, "CreateItem")
defer end()
_ = span
```

- `tracekit.B3(name, opts...)` / `tracekit.W3C(name, opts...)` return `kit.Tracer` for `rony.WithTracer`.
- `tracekit.NewSpan(ctx, name)` returns `(context.Context, trace.Span, func())`.
- Do not configure OTel SDK tracers by hand in feature code.
