# Devbox

Devbox installs optional infrastructure services into a **Kubernetes** cluster with **Helm**. Services include Postgres, Redis, and Temporal.

Use an existing kind, minikube, or cloud development cluster. You can also provision a local VM with Vagrant and microk8s.

## Prerequisites

Install these tools on the host:

- `kubectl`
- `helm`
- `yq`

For `cluster.mode: vagrant` in `config.yaml`, also install [Vagrant](https://developer.hashicorp.com/vagrant/install) and a provider. Supported providers include VirtualBox, UTM on Apple Silicon, and libvirt.

The default box is **`bento/ubuntu-24.04`**. Canonical no longer publishes `ubuntu/noble64` on Vagrant Cloud. Override `vm.box` in `config.yaml` when necessary.

```sh
# Pre-fetch the box (optional)
vagrant box add bento/ubuntu-24.04 --provider=virtualbox
```

Run `make bootstrap` to verify tools.

## Quick start with an existing cluster

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

You can also use the workspace Makefile:

```sh
make devbox-up
```

## Optional Vagrant and microk8s

Set these values in `config.yaml`:

```yaml
cluster:
  mode: vagrant
vm:
  box: bento/ubuntu-24.04   # default; change if your provider needs another box
```

`make up` provisions an Ubuntu 24.04 VM with microk8s and writes `shared/kubeconfig`.

On Apple Silicon, set the provider:

```sh
export VAGRANT_DEFAULT_PROVIDER=utm
```

## Makefile targets

| Target | `existing` cluster | `vagrant` mode |
|--------|--------------------|----------------|
| `bootstrap` | Verify kubectl, helm, and yq. Add chart repositories. | Also verify Vagrant |
| `up` | Install Helm releases | Start the VM and install releases |
| `down` | Remove devbox Helm releases | Stop the VM gracefully |
| `suspend` or `pause` | Not applicable | Save VM state with `vagrant suspend` |
| `resume` | Not applicable | Wake the suspended VM |
| `destroy` | Remove devbox Helm releases | Destroy the VM |
| `kubeconfig` | Not applicable | Refresh `shared/kubeconfig` |
| `services` | Sync Helm releases from `config.yaml` toggles | Same behavior |
| `status` | Run `kubectl get nodes` | Show Vagrant and kubectl status |

## Configuration

Edit `config.yaml`:

- **`cluster.mode`:** `existing` by default. Use `vagrant` for VM provisioning.
- **`cluster.kubeconfig`:** Optional path for an existing cluster.
- **`app.name`:** Application slug for local hostnames. The scaffold sets this value from the project name.
- **`dns.tld`:** Local DNS suffix. Default is `localdev`.
- **`vm`:** Vagrant VM sizing for `vagrant` mode.
- **`services`:** Boolean toggle for each platform component.

| Service | Default | Notes |
|---------|---------|-------|
| `postgres` | On | CloudNativePG cluster with `dbUser`, `dbPass`, and `user-db`. Installs the cnpg operator and a PVC. |
| `redis` | On | Redis-compatible DragonflyDB without authentication in development |
| `temporal` | Off | Temporal server and UI. Requires `postgres` and uses the same database credentials. |
| `redpanda` | Off | Single-node Redpanda |
| `observability` | Off | OTel Collector, Jaeger, and Grafana |
| `tigerbeetle` | Off | Raw StatefulSet manifest |
| `rustfs` | Off | Standalone RustFS S3-compatible object storage |

After toggle changes, run `make services`. For a cold start, run `make up`.

### Service credentials for development

Set credentials in `services/values/*.yaml`, not `config.yaml`. Change these values for your environment.

PostgreSQL is the exception. `scripts/services.sh` creates its user and password in the `postgres-app` secret. The username must match `cluster.initdb.owner` in `postgres.yaml`.

| Service | Authentication | Default | Values file |
|---------|----------------|---------|-------------|
| PostgreSQL | User, password, and database | `dbUser`, `dbPass`, `user-db` | `services/values/postgres.yaml` and `postgres-app` secret |
| DragonflyDB | Disabled | None | `services/values/dragonfly.yaml` |
| Temporal | Uses PostgreSQL | Same as PostgreSQL | `services/values/temporal.yaml` |
| Redpanda | TLS disabled | None | `services/values/redpanda.yaml` |
| Grafana | Admin user and password | `admin`, `admin` | `services/values/grafana.yaml` |
| Jaeger | In-memory storage | None | `services/values/jaeger.yaml` |
| OTel Collector | None | None | `services/values/otel-collector.yaml` |
| RustFS | Access key and secret key | `s3User`, `s3Pass` | `services/values/rustfs.yaml` |

`make services` registers only missing Helm chart repositories. `make bootstrap` refreshes repository indexes.

## Service endpoints in Vagrant mode

With `cluster.mode: vagrant`, devbox exposes enabled services on predictable hostnames:

| Service | Host with default app name `demo` | Port |
|---------|-----------------------------------|------|
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

### Hostname operation

1. `make bootstrap` installs the [vagrant-dns](https://github.com/BerlinVagrant/vagrant-dns) plugin. It registers the local TLD with `vagrant dns --install` and `vagrant dns --start`.
2. `make up` starts the VM. vagrant-dns resolves `*.<app>.localdev` to the VM IP. `Vagrantfile` defines the patterns.
3. `scripts/apply-exposure.sh` configures **Traefik**. HTTP uses `Ingress` routes. Database ports use `IngressRouteTCP` pass-through. `provision.sh` installs Traefik on the VM's host network. Each port binds to the VM IP.

After `make services`, the script prints active endpoints. Run `make dns` again if hostnames stop resolving.

### Existing cluster exposure

HTTP ingress uses your cluster's ingress class. TCP exposure uses Traefik `IngressRouteTCP` when the Traefik CRDs exist.

For nginx, TCP exposure uses config map `ingress/nginx-ingress-tcp-microk8s-conf` or `ingress-nginx/tcp-services`. Otherwise, use `kubectl port-forward`.

## Kubernetes access

**Existing cluster:** Use `cluster.kubeconfig`, `$KUBECONFIG`, or `~/.kube/config`.

**Vagrant mode:** The admin config is written to `shared/kubeconfig`, which Git ignores. The API endpoint is `https://127.0.0.1:16443`.

```sh
kubectl -n devbox get pods
```

### Reach services from the host

**Vagrant mode:** Use the `*.<app.name>.localdev` hostnames from the endpoint table. Port forwarding is not necessary.

**Existing cluster:** Use port forwarding when ingress TCP is unavailable:

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

- **Box not found with `ubuntu/noble64` 404:** Ubuntu stopped publishing official Noble boxes. Set `vm.box: bento/ubuntu-24.04` in `config.yaml`. Then run `vagrant box add bento/ubuntu-24.04 --provider=virtualbox`.
- **kubectl cannot connect after `make up`:** Run `make kubeconfig` again. If the failure continues, run `vagrant status`. The expected state is `running`.
- **Helm release fails:** Verify that the cluster has sufficient resources. Disable heavy services in `config.yaml`.
- **Helmfile or helm-diff errors:** Remove `services/helmfile.yaml` if it exists. Devbox uses `scripts/services.sh sync` through `make services`. No Helmfile plugin is necessary.
- **Temporal fails on cassandra key:** Verify that `services/values/temporal.yaml` uses `server.config.persistence.datastores`. Temporal chart v1.0 and later requires this setting. Run `make services` again.
- **Temporal requires Postgres:** Set `services.postgres: true` when you enable `services.temporal`. Temporal creates its own databases. `postgres.yaml` grants `CREATEDB` to the app role with `initdb.postInitSQL`.
- **Postgres pod stays `Pending`:** CloudNativePG always provisions a PVC. The cluster needs a default `StorageClass`. For microk8s, run `microk8s enable hostpath-storage`. Kind includes `local-path`. Check with `kubectl get sc`.
- **`postgresql.cnpg.io` CRD not found:** The cnpg operator installation failed or was skipped. Set `services.postgres: true` and run `make services` again. Verify with `kubectl -n cnpg-system get pods`.
- **Hostnames do not resolve:** Run `make dns` in Vagrant mode. If you changed `dns.tld`, run `make bootstrap` again.
- **TCP endpoint refused:** Check Traefik health with `kubectl -n traefik get pods`. Run `make services` again. A new TCP service also needs a matching entry point in `services/values/traefik.yaml`. Run `make up` after this edit to provision the VM again.
- **Vagrant VM does not start:** Check the provider with `vagrant status` and `VAGRANT_DEFAULT_PROVIDER`.
