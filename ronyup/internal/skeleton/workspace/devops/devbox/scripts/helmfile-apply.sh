#!/usr/bin/env bash
# Install or remove Helm releases according to config.yaml (no helmfile / helm-diff).
# Invoked by: install-services.sh (make services / make up).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/services"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

CONFIG="$ROOT/config.yaml"
NS=devbox

# service_enabled checks a boolean toggle under services.* in config.yaml.
service_enabled() {
  yq -e ".services.${1} == true" "$CONFIG" >/dev/null 2>&1
}

install_release() {
  local name="$1"
  local chart="$2"
  local values_file="$3"

  echo "==> installing $name ($chart)"
  helm upgrade --install "$name" "$chart" \
    --namespace "$NS" \
    --create-namespace \
    -f "$values_file" \
    --wait \
    --timeout 15m
}

# Best-effort uninstall; missing releases are ignored.
remove_release() {
  local name="$1"

  echo "==> removing $name"
  helm uninstall "$name" --namespace "$NS" 2>/dev/null || true
}

# Install or uninstall a single release based on its config.yaml toggle.
sync_release() {
  local config_key="$1"
  local release_name="$2"
  local chart="$3"
  local values_file="$4"

  if service_enabled "$config_key"; then
    install_release "$release_name" "$chart" "$values_file"
  else
    remove_release "$release_name"
  fi
}

helm_repo_ensure

# Per-service releases: install when enabled, uninstall when disabled.
sync_release postgres postgres bitnami/postgresql values/postgres.yaml
sync_release redis redis bitnami/redis values/redis.yaml
sync_release temporal temporal temporalio/temporal values/temporal.yaml
sync_release redpanda redpanda redpanda/redpanda values/redpanda.yaml

# Observability is a bundle of three charts controlled by one config toggle.
if service_enabled observability; then
  install_release otel-collector open-telemetry/opentelemetry-collector values/otel-collector.yaml
  install_release jaeger jaegertracing/jaeger values/jaeger.yaml
  install_release grafana grafana/grafana values/grafana.yaml
else
  remove_release grafana
  remove_release jaeger
  remove_release otel-collector
fi

echo "Helm releases synced"
