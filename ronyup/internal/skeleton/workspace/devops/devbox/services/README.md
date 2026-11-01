# Helm chart values for devbox services

`../scripts/services.sh sync` installs releases with plain `helm upgrade --install`. Helmfile and the helm-diff plugin are not used.

`../scripts/apply-exposure.sh` configures host exposure with `exposure.yaml`. It uses HTTP `Ingress` and Traefik `IngressRouteTCP` pass-through. On clusters running nginx, it falls back to an nginx TCP config map.

`../scripts/provision.sh` installs Traefik on the VM with `values/traefik.yaml`.

Toggle services in `../config.yaml`, then run `make services`.

## Local hostnames

Hostnames use `<host>.<app.name>.<dns.tld>`. Examples are `db.demo.localdev:5432` and `temporal.demo.localdev:7233`.

See `exposure.yaml` for the full host-to-Kubernetes-service map. Edit this file to add or rename endpoints.

## Credentials

Development defaults are in `values/*.yaml`:

| File | Settings |
|------|----------|
| `postgres.yaml` | `cluster.initdb.database` and `.owner`. `../scripts/services.sh` creates the owner password in the `postgres-app` secret. |
| `dragonfly.yaml` | No authentication in development. `storage.enabled: false`. |
| `temporal.yaml` | Reuses the PostgreSQL user and password. Connects in-cluster to `postgres-rw.devbox.svc.cluster.local`. |
| `grafana.yaml` | `adminUser`, `adminPassword` |
| `jaeger.yaml` | In-memory storage and no login |
| `redpanda.yaml` | TLS disabled for local development |
| `rustfs.yaml` | `secret.rustfs.access_key` and `.secret_key` in standalone mode |
| `otel-collector.yaml` | Exports traces to Jaeger in-cluster |

The [CloudNativePG](https://cloudnative-pg.io/) operator provisions PostgreSQL. `services.sh` installs the operator into `cnpg-system` and applies the `postgres-app` credentials secret. It then installs the `cnpg/cluster` chart with services `postgres-rw`, `-ro`, and `-r`.

DragonflyDB is a Redis-compatible store installed from its OCI chart.

Enable PostgreSQL in `config.yaml` when you enable Temporal. The Temporal chart does not include a database subchart.

RustFS is an S3-compatible object store. The [RustFS Helm chart](https://charts.rustfs.com/) installs it in standalone mode with one pod and PVC.

When exposure is configured, use `rustfs.<app>.localdev:9000` for the S3 API. Use `http://rustfs-console.<app>.localdev/` for the web console.
