# Devbox

Local development platform: install optional infrastructure services (Postgres, Redis, Temporal, …) into a **Kubernetes** cluster via **Helmfile**.

Use your existing cluster (kind, minikube, cloud dev cluster, …) or optionally provision a local VM with Vagrant + microk8s.

## Prerequisites

Always required on the host:

- `kubectl`, `helm`, `helmfile`, `yq`

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
| `bootstrap` | verify kubectl, helm, helmfile, yq | also verify vagrant |
| `up` | install Helmfile releases | start VM + install releases |
| `down` | remove devbox Helm releases | halt VM (graceful shutdown) |
| `suspend` / `pause` | n/a | save VM state (`vagrant suspend`) |
| `resume` | n/a | wake suspended VM |
| `destroy` | remove devbox Helm releases | destroy VM |
| `kubeconfig` | n/a | refresh `shared/kubeconfig` |
| `services` | sync Helmfile releases from `config.yaml` toggles | same |
| `status` | `kubectl get nodes` | vagrant + kubectl |

## Configuration

Edit `config.yaml`:

- **`cluster.mode`**: `existing` (default) or `vagrant`
- **`cluster.kubeconfig`**: optional path when using an existing cluster
- **`vm`**: Vagrant VM sizing (only for `vagrant` mode)
- **`services`**: boolean toggles per platform component

| Service | Default | Notes |
|---------|---------|-------|
| `postgres` | on | Bitnami PostgreSQL (`dbUser` / `dbPass` / `user-db`) |
| `redis` | on | Bitnami Redis (no auth in dev) |
| `temporal` | off | Temporal server + bundled Postgres |
| `redpanda` | off | Single-node Redpanda |
| `observability` | off | OTel Collector → Jaeger + Grafana |
| `tigerbeetle` | off | Raw manifest (StatefulSet) |

After changing toggles, run `make services` (or `make up` from a cold start).

## Kubernetes access

**Existing cluster:** uses your kubeconfig (`cluster.kubeconfig`, `$KUBECONFIG`, or `~/.kube/config`).

**Vagrant mode:** admin config is written to `shared/kubeconfig` (gitignored), API on `https://127.0.0.1:16443`.

```sh
kubectl -n devbox get pods
```

### Reach services from the host

Port-forward as needed:

```sh
kubectl -n devbox port-forward svc/postgres-postgresql 5432:5432
kubectl -n devbox port-forward svc/redis-master 6379:6379
kubectl -n devbox port-forward svc/grafana 3000:80
kubectl -n devbox port-forward svc/jaeger-query 16686:16686
kubectl -n devbox port-forward svc/otel-collector-opentelemetry-collector 4317:4317
```

## Layout

```
devbox/
├── config.yaml          # cluster mode + service toggles
├── Vagrantfile          # only used when cluster.mode=vagrant
├── Makefile
├── scripts/
├── services/            # Helmfile + values
└── shared/              # kubeconfig (vagrant mode, gitignored)
```

## Troubleshooting

- **Box not found (`ubuntu/noble64` 404)**: Ubuntu stopped publishing official Noble boxes. Set `vm.box: bento/ubuntu-24.04` in `config.yaml` (the scaffold default) and run `vagrant box add bento/ubuntu-24.04 --provider=virtualbox`.
- **kubectl can't connect after `make up`**: the VM is usually running — re-run `make kubeconfig`. If it still fails, `vagrant status` should show `running`.
- **Helm release fails**: ensure the cluster has enough resources; disable heavy services in `config.yaml`.
- **Vagrant VM won't start**: check provider (`vagrant status`, `VAGRANT_DEFAULT_PROVIDER`).
