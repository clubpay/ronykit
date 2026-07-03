# Helm chart values for devbox services

Releases are installed by `../scripts/services.sh sync` using plain `helm upgrade --install` (no Helmfile or helm-diff plugin).

Host exposure is configured by `../scripts/apply-exposure.sh` using `exposure.yaml` (HTTP `Ingress` + Traefik `IngressRouteTCP` passthrough; falls back to an nginx TCP configmap on clusters running nginx). Traefik itself is installed on the VM by `../scripts/provision.sh` from `values/traefik.yaml`.

Toggle services in `../config.yaml`, then run `make services`.

## Local hostnames

Pattern: `<host>.<app.name>.<dns.tld>` (defaults: `db.demo.localdev:5432`, `temporal.demo.localdev:7233`, …).

See `exposure.yaml` for the full host → Kubernetes service map. Edit there to add or rename endpoints.

## Credentials

Dev defaults live in `values/*.yaml`:

| File | Settings |
|------|----------|
| `postgres.yaml` | `cluster.initdb.database` / `.owner`; the owner password lives in the `postgres-app` secret (created by `../scripts/services.sh`) |
| `dragonfly.yaml` | no auth in dev (`storage.enabled: false`) |
| `temporal.yaml` | Reuses PostgreSQL user/password; connects in-cluster to `postgres-rw.devbox.svc.cluster.local` |
| `grafana.yaml` | `adminUser`, `adminPassword` |
| `jaeger.yaml` | In-memory storage, no login |
| `redpanda.yaml` | TLS disabled for local dev |
| `rustfs.yaml` | `secret.rustfs.access_key` / `.secret_key`; standalone mode |
| `otel-collector.yaml` | Exports traces to Jaeger in-cluster |

PostgreSQL is provisioned by the [CloudNativePG](https://cloudnative-pg.io/) operator: `services.sh` installs the operator into `cnpg-system`, applies the `postgres-app` credentials secret, then installs the `cnpg/cluster` chart (services `postgres-rw` / `-ro` / `-r`). DragonflyDB is a Redis-compatible store installed from its OCI chart.

When enabling Temporal, also enable PostgreSQL in `config.yaml` — the chart no longer bundles a database sub-chart.

RustFS is an S3-compatible object store installed from the [RustFS Helm chart](https://charts.rustfs.com/) in standalone mode (single pod + PVC). Use `rustfs.<app>.localdev:9000` for the S3 API and `http://rustfs-console.<app>.localdev/` for the web console when exposure is configured.
