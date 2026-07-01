#!/usr/bin/env bash
# Uninstall every devbox Helm release (regardless of config.yaml toggles).
# Invoked by: make down / make destroy (existing cluster mode via down.sh).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/services"

NS=devbox

# Order does not matter; ignore releases that were never installed.
for release in grafana jaeger otel-collector redpanda temporal redis postgres; do
  helm uninstall "$release" --namespace "$NS" 2>/dev/null || true
done

echo "Helm releases removed"
