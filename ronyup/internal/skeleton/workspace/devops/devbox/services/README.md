# Helm chart values for devbox services

Releases are installed by `../scripts/helmfile-apply.sh` using plain `helm upgrade --install` (no Helmfile or helm-diff plugin).

Toggle services in `../config.yaml`, then run `make services`.

## Credentials

Dev defaults live in `values/*.yaml`:

| File | Settings |
|------|----------|
| `postgres.yaml` | `auth.username`, `auth.password`, `auth.database` |
| `redis.yaml` | `auth.enabled: false` |
| `temporal.yaml` | Reuses PostgreSQL user/password; connects to `postgres-postgresql.devbox.svc.cluster.local` |
| `grafana.yaml` | `adminUser`, `adminPassword` |
| `jaeger.yaml` | In-memory storage, no login |
| `redpanda.yaml` | TLS disabled for local dev |
| `otel-collector.yaml` | Exports traces to Jaeger in-cluster |

When enabling Temporal, also enable PostgreSQL in `config.yaml` — the chart no longer bundles a database sub-chart.
