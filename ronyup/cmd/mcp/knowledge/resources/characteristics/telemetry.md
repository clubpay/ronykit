---
keywords:
  - telemetry
  - observ
  - log
  - trac
  - metric
applies_to_files:
  - app
  - api
  - repo
---
Instrument with x/telemetry: use logkit for structured logging, tracekit for distributed traces, and meterkit for Prometheus metrics.

## File-Level Hint

Inject x/telemetry/logkit.Logger and add tracekit spans for observable behavior.
