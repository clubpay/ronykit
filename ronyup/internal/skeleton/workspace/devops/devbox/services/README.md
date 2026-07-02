# Helm chart values for devbox services

Releases are installed by `../scripts/services.sh sync` using plain `helm upgrade --install` (no Helmfile or helm-diff plugin).

Host exposure is configured by `../scripts/apply-exposure.sh` using `exposure.yaml` (HTTP ingress + nginx TCP passthrough).

Toggle services in `../config.yaml`, then run `make services`.

## Local hostnames

Pattern: `<host>.<app.name>.<dns.tld>` (defaults: `db.demo.localdev:5432`, `temporal.demo.localdev:7233`, …).

See `exposure.yaml` for the full host → Kubernetes service map. Edit there to add or rename endpoints.

## Credentials

Dev defaults live in `values/*.yaml`:

| File | Settings |
|------|----------|
| `postgres.yaml` | `auth.username`, `auth.password`, `auth.database` |
| `redis.yaml` | `auth.enabled: false` |
| `temporal.yaml` | Reuses PostgreSQL user/password; connects in-cluster to `postgres-postgresql.devbox.svc.cluster.local` |
| `grafana.yaml` | `adminUser`, `adminPassword` |
| `jaeger.yaml` | In-memory storage, no login |
| `redpanda.yaml` | TLS disabled for local dev |
| `otel-collector.yaml` | Exports traces to Jaeger in-cluster |

When enabling Temporal, also enable PostgreSQL in `config.yaml` — the chart no longer bundles a database sub-chart.
