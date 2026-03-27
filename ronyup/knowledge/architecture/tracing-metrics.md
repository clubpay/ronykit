Trace with `x/telemetry/tracekit` and measure with `x/telemetry/meterkit`.

- Use `tracekit.W3C()` or `tracekit.B3()` as the `kit.Tracer`.
- Expose metrics via `meterkit.WithPrometheus`.
