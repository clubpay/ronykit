# Devbox

Local development platform: install optional infrastructure services (Postgres, Redis, Temporal, …) into a **Kubernetes** cluster via **Helm**.

Use your existing cluster (kind, minikube, cloud dev cluster, …) or optionally provision a local VM with Vagrant + microk8s.

## Prerequisites

Always required on the host:

- `kubectl`, `helm`, `yq`

When `cluster.mode: vagrant` in `config.yaml`, you also need [Vagrant](https://developer.hashicorp.com/vagrant/install) and a provider (VirtualBox, UTM on Apple Silicon, libvirt, …).

The default box is **`bento/ubuntu-24.04`** (Canonical no longer publishes `ubuntu/noble64` on Vagrant Cloud). Override `vm.box` in `config.yaml` if needed.

```sh
# Pre-fetch the box (optional)
vagrant box add bento/ubuntu-24.04 --provider=virtualbox
```

Run `make bootstrap` to verify tools.

## Quick start (existing cluster)

```sh
cd devops/devbox

# 1. Point at your cluster (default: $KUBECONFIG or ~/.kube/config)
export KUBECONFIG=~/.kube/config   # optional if already set

# 2. Opt in/out of services
$EDITOR config.yaml                 # cluster.mode: existing

# 3. Install enabled services into namespace devbox
make up
kubectl -n devbox get pods
```

From the workspace Makefile:

```sh
make devbox-up
```

## Optional: Vagrant + microk8s

Set in `config.yaml`:

```yaml
cluster:
  mode: vagrant
vm:
  box: bento/ubuntu-24.04   # default; change if your provider needs another box
```

Then `make up` provisions an Ubuntu 24.04 VM with microk8s and writes `shared/kubeconfig`. On Apple Silicon:

```sh
export VAGRANT_DEFAULT_PROVIDER=utm
```

## Makefile targets

| Target | `existing` cluster | `vagrant` mode |
|--------|-------------------|----------------|
| `bootstrap` | verify kubectl, helm, yq; add chart repos | also verify vagrant |
| `up` | install Helm releases | start VM + install releases |
| `down` | remove devbox Helm releases | halt VM (graceful shutdown) |
| `suspend` / `pause` | n/a | save VM state (`vagrant suspend`) |
| `resume` | n/a | wake suspended VM |
| `destroy` | remove devbox Helm releases | destroy VM |
| `kubeconfig` | n/a | refresh `shared/kubeconfig` |
| `services` | sync Helm releases from `config.yaml` toggles | same |
| `status` | `kubectl get nodes` | vagrant + kubectl |

## Configuration

Edit `config.yaml`:

- **`cluster.mode`**: `existing` (default) or `vagrant`
- **`cluster.kubeconfig`**: optional path when using an existing cluster
- **`app.name`**: application slug used in local hostnames (scaffold sets this from your project name)
- **`dns.tld`**: local DNS suffix (default `localdev`)
- **`vm`**: Vagrant VM sizing (only for `vagrant` mode)
- **`services`**: boolean toggles per platform component

| Service | Default | Notes |
|---------|---------|-------|
| `postgres` | on | CloudNativePG cluster (`dbUser` / `dbPass` / `user-db`); installs the cnpg operator + a PVC |
| `redis` | on | DragonflyDB, Redis-compatible (no auth in dev) |
| `temporal` | off | Temporal server + UI; requires `postgres` (uses same DB credentials) |
| `redpanda` | off | Single-node Redpanda |
| `observability` | off | OTel Collector → Jaeger + Grafana |
| `tigerbeetle` | off | Raw manifest (StatefulSet) |
| `rustfs` | off | RustFS S3-compatible object storage (standalone) |

After changing toggles, run `make services` (or `make up` from a cold start).

### Service credentials (dev defaults)

Credentials are set in `services/values/*.yaml` (not in `config.yaml`). Change them there for your environment. PostgreSQL is the exception: its user/password live in the `postgres-app` secret created by `scripts/services.sh` (username must match `cluster.initdb.owner` in `postgres.yaml`).

| Service | Auth | Default | Values file |
|---------|------|---------|-------------|
| PostgreSQL | user / password / database | `dbUser` / `dbPass` / `user-db` | `services/values/postgres.yaml` (+ `postgres-app` secret) |
| DragonflyDB | none (auth disabled) | — | `services/values/dragonfly.yaml` |
| Temporal | uses PostgreSQL above | same as PostgreSQL | `services/values/temporal.yaml` |
| Redpanda | none (TLS disabled) | — | `services/values/redpanda.yaml` |
| Grafana | admin user / password | `admin` / `admin` | `services/values/grafana.yaml` |
| Jaeger | none (in-memory storage) | — | `services/values/jaeger.yaml` |
| OTel Collector | none | — | `services/values/otel-collector.yaml` |
| RustFS | access key / secret key | `s3User` / `s3Pass` | `services/values/rustfs.yaml` |

`make services` only registers missing Helm chart repos; `make bootstrap` refreshes repo indexes.

## Service endpoints (vagrant mode)

With `cluster.mode: vagrant`, devbox exposes enabled services on predictable hostnames:

| Service | Host (default app name `demo`) | Port |
|---------|--------------------------------|------|
| PostgreSQL | `db.demo.localdev` | 5432 |
| Redis | `redis.demo.localdev` | 6379 |
| Temporal gRPC | `temporal.demo.localdev` | 7233 |
| Temporal UI | `http://temporal-ui.demo.localdev/` | 80 |
| Grafana | `http://grafana.demo.localdev/` | 80 |
| Jaeger | `http://jaeger.demo.localdev/` | 80 |
| OTel gRPC | `otel.demo.localdev` | 4317 |
| Redpanda Kafka | `redpanda.demo.localdev` | 9092 |
| RustFS S3 API | `rustfs.demo.localdev` | 9000 |
| RustFS console | `http://rustfs-console.demo.localdev/` | 80 |

Replace `demo` with your `app.name` from `config.yaml`.

**How it works**

1. `make bootstrap` installs the [vagrant-dns](https://github.com/BerlinVagrant/vagrant-dns) plugin and registers the local TLD on your host (`vagrant dns --install` / `vagrant dns --start`).
2. `make up` starts the VM; vagrant-dns resolves `*.<app>.localdev` to the VM IP (patterns in `Vagrantfile`).
3. `scripts/apply-exposure.sh` configures **Traefik** (HTTP `Ingress` routes + `IngressRouteTCP` passthrough for database ports). Traefik is installed by `provision.sh` and runs on the VM's host network, so each port binds on the VM IP.

After `make services`, the script prints the active endpoints. Re-run `make dns` if hostnames stop resolving.

**Existing cluster mode:** HTTP ingress is applied against your cluster's ingress class. TCP exposure uses Traefik `IngressRouteTCP` when the Traefik CRDs are present, or an nginx TCP configmap (`ingress/nginx-ingress-tcp-microk8s-conf` or `ingress-nginx/tcp-services`) if you run nginx. Use `kubectl port-forward` otherwise (see below).

## Kubernetes access

**Existing cluster:** uses your kubeconfig (`cluster.kubeconfig`, `$KUBECONFIG`, or `~/.kube/config`).

**Vagrant mode:** admin config is written to `shared/kubeconfig` (gitignored), API on `https://127.0.0.1:16443`.

```sh
kubectl -n devbox get pods
```

### Reach services from the host

**Vagrant mode:** use the `*.<app.name>.localdev` hostnames from the table above (no port-forward needed).

**Existing cluster** (or when ingress TCP is unavailable), port-forward as needed:

```sh
kubectl -n devbox port-forward svc/postgres-rw 5432:5432
kubectl -n devbox port-forward svc/dragonfly 6379:6379
kubectl -n devbox port-forward svc/temporal-web 8080:8080
kubectl -n devbox port-forward svc/grafana 3000:80
kubectl -n devbox port-forward svc/jaeger-query 16686:16686
kubectl -n devbox port-forward svc/otel-collector-opentelemetry-collector 4317:4317
kubectl -n devbox port-forward svc/rustfs-svc 9000:9000
kubectl -n devbox port-forward svc/rustfs-svc 9001:9001
```

## Layout

```
devbox/
├── config.yaml          # cluster mode + app/dns + service toggles
├── Vagrantfile          # only used when cluster.mode=vagrant
├── Makefile
├── scripts/
│   ├── cluster.sh       # VM/cluster lifecycle (up, down, suspend, resume, destroy)
│   ├── services.sh      # Helm releases + exposure (sync, remove)
│   └── …                # bootstrap, dns, kubeconfig, provision, …
├── services/            # Helm values, exposure.yaml, manifests/
└── shared/              # kubeconfig (vagrant mode, gitignored)
```

## Troubleshooting

- **Box not found (`ubuntu/noble64` 404)**: Ubuntu stopped publishing official Noble boxes. Set `vm.box: bento/ubuntu-24.04` in `config.yaml` (the scaffold default) and run `vagrant box add bento/ubuntu-24.04 --provider=virtualbox`.
- **kubectl can't connect after `make up`**: the VM is usually running — re-run `make kubeconfig`. If it still fails, `vagrant status` should show `running`.
- **Helm release fails**: ensure the cluster has enough resources; disable heavy services in `config.yaml`.
- **Helmfile / helm-diff errors**: remove `services/helmfile.yaml` if present — devbox uses `scripts/services.sh sync` via `make services` (no Helmfile plugin).
- **Temporal fails on cassandra key**: ensure `services/values/temporal.yaml` uses `server.config.persistence.datastores` (Temporal chart v1.0+). Re-run `make services`.
- **Temporal requires postgres**: set `services.postgres: true` when enabling `services.temporal`. Temporal creates its own databases; `postgres.yaml` grants the app role `CREATEDB` via `initdb.postInitSQL` so this works out of the box.
- **Postgres pod stuck `Pending`**: CloudNativePG always provisions a PVC — the cluster needs a default `StorageClass` (microk8s: `microk8s enable hostpath-storage`; kind ships `local-path`). Check with `kubectl get sc`.
- **`postgresql.cnpg.io` CRD not found**: the cnpg operator install failed or was skipped. Re-run `make services` with `services.postgres: true`; verify `kubectl -n cnpg-system get pods`.
- **Hostnames do not resolve**: run `make dns` (vagrant mode). Re-run `make bootstrap` if you change `dns.tld` in `config.yaml`.
- **TCP endpoint refused**: check Traefik is healthy (`kubectl -n traefik get pods`) and re-run `make services`. Adding a new TCP service also needs a matching entrypoint in `services/values/traefik.yaml` — re-provision the VM (`make up`) after editing it.
- **Vagrant VM won't start**: check provider (`vagrant status`, `VAGRANT_DEFAULT_PROVIDER`).
