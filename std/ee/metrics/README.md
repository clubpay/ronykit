# ee metrics

OpenTelemetry instruments for an [`ee`](../../../../ee) `Env`. The meter comes
from [`x/telemetry/meterkit`](../../../../x/telemetry/meterkit).

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

`ee.runtime.signals` counts observations. The `kind` attribute is an `ee.Obs*`
constant. Outbox publish and outbox error add the batch size; other kinds add
one.

`ee.outbox.pending` and `ee.outbox.oldest_age_seconds` come from `RecordStats`.
A store that cannot report a backlog (`OutboxPending < 0`) does not move the
pending gauge.
