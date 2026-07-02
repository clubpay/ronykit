#!/usr/bin/env bash
# Devbox Kubernetes service management.
#
# Usage: services.sh <command>
#   sync    - install, upgrade, or remove releases to match config.yaml toggles
#   remove  - uninstall all devbox Helm releases, exposure, and raw manifests
#
# Invoked by: make services, cluster.sh up/down/destroy
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

readonly DEVBOX_NS=devbox
readonly SERVICES_DIR="$ROOT/services"

# CloudNativePG operator (cluster-scoped; installs CRDs into its own namespace).
readonly CNPG_NS=cnpg-system
readonly CNPG_OPERATOR_RELEASE=cnpg
# DragonflyDB ships as an OCI chart (no classic repo). Pinned for reproducible
# installs — bump this tag to upgrade; set empty to track the latest chart.
readonly DRAGONFLY_CHART="oci://ghcr.io/dragonflydb/dragonfly/helm/dragonfly"
readonly DRAGONFLY_VERSION="v1.39.0"

usage() {
  cat <<EOF
Usage: $(basename "$0") <command>

Commands:
  sync     install, upgrade, or remove releases to match config.yaml toggles
  remove   uninstall all devbox Helm releases, exposure rules, and manifests
  help     show this help
EOF
}

helm_install_release() {
  local name="$1"
  local chart="$2"
  local values_file="$3"

  echo "==> installing $name ($chart)"
  helm upgrade --install "$name" "$chart" \
    --namespace "$DEVBOX_NS" \
    --create-namespace \
    -f "$values_file" \
    --wait \
    --timeout 15m
}

helm_remove_release() {
  local name="$1"
  local ns="${2:-$DEVBOX_NS}"

  echo "==> removing $name"
  helm uninstall "$name" --namespace "$ns" 2>/dev/null || true
}

helm_sync_release() {
  local config_key="$1"
  local release_name="$2"
  local chart="$3"
  local values_file="$4"

  if service_enabled "$ROOT" "$config_key"; then
    helm_install_release "$release_name" "$chart" "$values_file"
  else
    helm_remove_release "$release_name"
  fi
}

# sync_postgres installs the CloudNativePG operator, the app credentials secret,
# and the Postgres Cluster (or tears the cluster + operator down when disabled).
sync_postgres() {
  if service_enabled "$ROOT" postgres; then
    echo "==> installing CloudNativePG operator ($CNPG_OPERATOR_RELEASE)"
    helm upgrade --install "$CNPG_OPERATOR_RELEASE" cnpg/cloudnative-pg \
      --namespace "$CNPG_NS" \
      --create-namespace \
      --wait \
      --timeout 15m

    echo "==> applying postgres credentials secret (postgres-app)"
    kubectl create secret generic postgres-app \
      --namespace "$DEVBOX_NS" \
      --type=kubernetes.io/basic-auth \
      --from-literal=username=dbUser \
      --from-literal=password=dbPass \
      --dry-run=client -o yaml | kubectl apply -f -

    helm_install_release postgres cnpg/cluster values/postgres.yaml
  else
    helm_remove_release postgres
    helm_remove_release "$CNPG_OPERATOR_RELEASE" "$CNPG_NS"
  fi
}

# sync_dragonfly installs the Redis-compatible DragonflyDB store (OCI chart).
sync_dragonfly() {
  if service_enabled "$ROOT" redis; then
    local -a args=(
      upgrade --install dragonfly "$DRAGONFLY_CHART"
      --namespace "$DEVBOX_NS"
      --create-namespace
      -f values/dragonfly.yaml
      --wait
      --timeout 15m
    )
    [[ -n "$DRAGONFLY_VERSION" ]] && args+=(--version "$DRAGONFLY_VERSION")

    echo "==> installing dragonfly ($DRAGONFLY_CHART)"
    helm "${args[@]}"
  else
    helm_remove_release dragonfly
  fi
}

require_postgres_for_temporal() {
  if service_enabled "$ROOT" temporal && ! service_enabled "$ROOT" postgres; then
    echo "services.temporal requires services.postgres (Temporal uses the devbox PostgreSQL release)" >&2
    exit 1
  fi
}

helm_sync_all() {
  cd "$SERVICES_DIR"
  helm_repo_ensure

  sync_postgres
  sync_dragonfly
  require_postgres_for_temporal
  helm_sync_release temporal temporal temporalio/temporal values/temporal.yaml
  helm_sync_release redpanda redpanda redpanda/redpanda values/redpanda.yaml

  if service_enabled "$ROOT" observability; then
    helm_install_release otel-collector open-telemetry/opentelemetry-collector values/otel-collector.yaml
    helm_install_release jaeger jaegertracing/jaeger values/jaeger.yaml
    helm_install_release grafana grafana/grafana values/grafana.yaml
  else
    helm_remove_release grafana
    helm_remove_release jaeger
    helm_remove_release otel-collector
  fi

  echo "Helm releases synced"
}

helm_remove_all() {
  local release

  cd "$SERVICES_DIR"

  for release in grafana jaeger otel-collector redpanda temporal dragonfly postgres; do
    helm_remove_release "$release"
  done
  helm_remove_release "$CNPG_OPERATOR_RELEASE" "$CNPG_NS"

  echo "Helm releases removed"
}

sync_tigerbeetle_manifest() {
  if service_enabled "$ROOT" tigerbeetle; then
    kubectl apply -f "$SERVICES_DIR/manifests/tigerbeetle.yaml"
  else
    kubectl delete -f "$SERVICES_DIR/manifests/tigerbeetle.yaml" --ignore-not-found
  fi
}

remove_tigerbeetle_manifest() {
  if service_enabled "$ROOT" tigerbeetle; then
    kubectl delete -f "$SERVICES_DIR/manifests/tigerbeetle.yaml" --ignore-not-found
  fi
}

remove_exposure() {
  kubectl delete ingress devbox-exposure-http -n "$DEVBOX_NS" --ignore-not-found

  # Traefik TCP routes (present only when the Traefik CRDs are installed).
  if kubectl get crd ingressroutetcps.traefik.io >/dev/null 2>&1; then
    kubectl delete ingressroutetcp -n "$DEVBOX_NS" \
      -l app.kubernetes.io/component=exposure --ignore-not-found >/dev/null 2>&1 || true
  fi

  if kubectl get configmap nginx-ingress-tcp-microk8s-conf -n ingress >/dev/null 2>&1; then
    kubectl patch configmap nginx-ingress-tcp-microk8s-conf -n ingress \
      --type merge -p '{"data":{}}' >/dev/null 2>&1 || true
  fi

  if kubectl get configmap tcp-services -n ingress-nginx >/dev/null 2>&1; then
    kubectl patch configmap tcp-services -n ingress-nginx \
      --type merge -p '{"data":{}}' >/dev/null 2>&1 || true
  fi
}

cmd_sync() {
  export_kubeconfig_env "$ROOT"
  devbox_script "$ROOT" wait-k8s.sh

  kubectl create namespace "$DEVBOX_NS" --dry-run=client -o yaml | kubectl apply -f -

  helm_sync_all
  devbox_script "$ROOT" apply-exposure.sh
  sync_tigerbeetle_manifest

  echo "Devbox services applied (see config.yaml for enabled set)"
}

cmd_remove() {
  export_kubeconfig_env "$ROOT"

  helm_remove_all
  remove_exposure
  remove_tigerbeetle_manifest

  echo "Devbox services removed from cluster"
}

main() {
  local cmd="${1:-}"

  case "$cmd" in
    sync)
      cmd_sync
      ;;
    remove)
      cmd_remove
      ;;
    help | -h | --help)
      usage
      ;;
    "")
      echo "missing command" >&2
      usage >&2
      exit 1
      ;;
    *)
      echo "unknown command: $cmd" >&2
      usage >&2
      exit 1
      ;;
  esac
}

main "$@"
