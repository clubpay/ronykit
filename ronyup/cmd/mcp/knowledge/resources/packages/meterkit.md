---
import_path: github.com/clubpay/ronykit/x/telemetry/meterkit
short_name: meterkit
---
OpenTelemetry MeterProvider with optional Prometheus HTTP exposure.

## Usage Hint

Use meterkit.WithPrometheus to expose /metrics and meterkit.NewExporter for OTLP export.
