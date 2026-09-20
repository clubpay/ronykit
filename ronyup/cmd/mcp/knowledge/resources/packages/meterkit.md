---
import_path: github.com/clubpay/ronykit/x/telemetry/meterkit
short_name: meterkit
---

OpenTelemetry MeterProvider with optional Prometheus HTTP exposure. Use this instead of a raw OTel meter setup.

## Usage Hint

```go
exp, err := meterkit.NewExporter(
	meterkit.WithPrometheus("/metrics", 9090),
)
```

- `meterkit.NewExporter(opt ...ExporterOption) (*Exporter, error)`.
- `meterkit.WithPrometheus(path, port, opt...)` exposes Prometheus scrape.
- Wire from `pkg/runner`, not from a feature handler.
