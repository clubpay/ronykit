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

  echo "==> removing $name"
  helm uninstall "$name" --namespace "$DEVBOX_NS" 2>/dev/null || true
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

require_postgres_for_temporal() {
  if service_enabled "$ROOT" temporal && ! service_enabled "$ROOT" postgres; then
    echo "services.temporal requires services.postgres (Temporal uses the devbox PostgreSQL release)" >&2
    exit 1
  fi
}

helm_sync_all() {
  cd "$SERVICES_DIR"
  helm_repo_ensure

  helm_sync_release postgres postgres bitnami/postgresql values/postgres.yaml
  helm_sync_release redis redis bitnami/redis values/redis.yaml
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

  for release in grafana jaeger otel-collector redpanda temporal redis postgres; do
    helm_remove_release "$release"
  done

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
