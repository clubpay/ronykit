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
| `postgres` | on | Bitnami PostgreSQL (`dbUser` / `dbPass` / `user-db`) |
| `redis` | on | Bitnami Redis (no auth in dev) |
| `temporal` | off | Temporal server + UI; requires `postgres` (uses same DB credentials) |
| `redpanda` | off | Single-node Redpanda |
| `observability` | off | OTel Collector → Jaeger + Grafana |
| `tigerbeetle` | off | Raw manifest (StatefulSet) |

After changing toggles, run `make services` (or `make up` from a cold start).

### Service credentials (dev defaults)

Credentials are set in `services/values/*.yaml` (not in `config.yaml`). Change them there for your environment.

| Service | Auth | Default | Values file |
|---------|------|---------|-------------|
| PostgreSQL | user / password / database | `dbUser` / `dbPass` / `user-db` | `services/values/postgres.yaml` |
| Redis | none (auth disabled) | — | `services/values/redis.yaml` |
| Temporal | uses PostgreSQL above | same as PostgreSQL | `services/values/temporal.yaml` |
| Redpanda | none (TLS disabled) | — | `services/values/redpanda.yaml` |
| Grafana | admin user / password | `admin` / `admin` | `services/values/grafana.yaml` |
| Jaeger | none (in-memory storage) | — | `services/values/jaeger.yaml` |
| OTel Collector | none | — | `services/values/otel-collector.yaml` |

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

Replace `demo` with your `app.name` from `config.yaml`.

**How it works**

1. `make bootstrap` installs the [vagrant-dns](https://github.com/bergsland/vagrant-dns) plugin (vagrant mode).
2. `scripts/provision.sh` runs **dnsmasq** on the VM so `*.<app>.localdev` resolves to the VM IP.
3. `make up` runs `scripts/sync-dns.sh` to point your host at the VM DNS (macOS `/etc/resolver/<app>.localdev`, or `/etc/hosts` fallback).
4. `scripts/apply-exposure.sh` configures microk8s **nginx ingress** (HTTP routes + TCP passthrough for database ports).

After `make services`, the script prints the active endpoints. Re-run `make dns` if you add services or change `app.name`.

**Existing cluster mode:** ingress/TCP exposure is applied when your cluster has nginx ingress with a TCP configmap (`ingress/nginx-ingress-tcp-microk8s-conf` or `ingress-nginx/tcp-services`). Use `kubectl port-forward` otherwise (see below).

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
kubectl -n devbox port-forward svc/postgres-postgresql 5432:5432
kubectl -n devbox port-forward svc/redis-master 6379:6379
kubectl -n devbox port-forward svc/temporal-web 8080:8080
kubectl -n devbox port-forward svc/grafana 3000:80
kubectl -n devbox port-forward svc/jaeger-query 16686:16686
kubectl -n devbox port-forward svc/otel-collector-opentelemetry-collector 4317:4317
```

## Layout

```
devbox/
├── config.yaml          # cluster mode + app/dns + service toggles
├── Vagrantfile          # only used when cluster.mode=vagrant
├── Makefile
├── scripts/
├── services/            # Helm values, exposure.yaml, manifests/
└── shared/              # kubeconfig (vagrant mode, gitignored)
```

## Troubleshooting

- **Box not found (`ubuntu/noble64` 404)**: Ubuntu stopped publishing official Noble boxes. Set `vm.box: bento/ubuntu-24.04` in `config.yaml` (the scaffold default) and run `vagrant box add bento/ubuntu-24.04 --provider=virtualbox`.
- **kubectl can't connect after `make up`**: the VM is usually running — re-run `make kubeconfig`. If it still fails, `vagrant status` should show `running`.
- **Helm release fails**: ensure the cluster has enough resources; disable heavy services in `config.yaml`.
- **Helmfile / helm-diff errors**: remove `services/helmfile.yaml` if present — devbox uses `scripts/helmfile-apply.sh` via `make services` (no Helmfile plugin).
- **Temporal fails on cassandra key**: ensure `services/values/temporal.yaml` uses `server.config.persistence.datastores` (Temporal chart v1.0+). Re-run `make services`.
- **Temporal requires postgres**: set `services.postgres: true` when enabling `services.temporal`.
- **Hostnames do not resolve**: run `make dns` (vagrant mode). On macOS this writes `/etc/resolver/<app>.localdev`; Linux falls back to `/etc/hosts`.
- **TCP endpoint refused**: ensure microk8s ingress is enabled (`microk8s enable ingress`) and re-run `make services`.
- **Vagrant VM won't start**: check provider (`vagrant status`, `VAGRANT_DEFAULT_PROVIDER`).
