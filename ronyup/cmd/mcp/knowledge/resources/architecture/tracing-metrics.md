Trace with `x/telemetry/tracekit` and measure with `x/telemetry/meterkit`.

- Wire the gateway tracer via `rony.WithTracer(tracekit.B3("<service-name>"))`
  (or `tracekit.W3C(...)`) when building the `rony.Server`.
- Initialize the OTLP/terminal exporter at startup with
  `tracekit.NewExporter(name, opts...)` and bind its `Shutdown` to the fx
  lifecycle so traces flush on stop.
- Use `tracekit.Span(ctx)` inside middleware/handlers to enrich spans with
  attributes, and `tracekit.Event` for span events (the scaffold's `logMW` is a
  reference implementation).
- For metrics, call `meterkit.NewExporter(meterkit.WithName(...),
meterkit.WithPrometheus(path, port))` once at startup; use
  `exp.SetAsGlobal()` so application code can read the global meter provider.
