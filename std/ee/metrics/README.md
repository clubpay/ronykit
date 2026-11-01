# ee metrics

This module provides OpenTelemetry instruments for an [`ee`](../../../ee) `Env`. [`x/telemetry/meterkit`](../../../x/telemetry/meterkit) supplies the meter.

```go
exp, err := meterkit.NewExporter(meterkit.WithPrometheus("/metrics", 9464))
rec, err := metrics.New(exp.Meter("ee"))

env, err := ee.New(
    ee.WithStore(s),
    ee.WithBus(b),
    rec.Option(),
)

// On a timer, so the outbox gauges move even when nothing else is recorded.
stats, err := env.Stats(ctx)
rec.RecordStats(ctx, stats)
```

`ee.runtime.signals` counts observations. The `kind` attribute contains an `ee.Obs*` constant. Outbox publish and outbox error add the batch size. Other kinds add one.

`RecordStats` supplies `ee.outbox.pending` and `ee.outbox.oldest_age_seconds`. A store can report `OutboxPending < 0` when it cannot report a backlog. In this case, the pending gauge does not change.
