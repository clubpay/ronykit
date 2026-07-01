#!/usr/bin/env bash
# Apply or remove Helmfile releases according to config.yaml service toggles.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/services"

CONFIG="$ROOT/config.yaml"
HF=(helmfile -f helmfile.yaml)

service_enabled() {
  yq -e ".services.${1} == true" "$CONFIG" >/dev/null 2>&1
}

sync_release() {
  local config_key="$1"
  local label="$2"

  if service_enabled "$config_key"; then
    echo "==> applying devbox-service=$label"
    "${HF[@]}" -l "devbox-service=$label" apply
  else
    echo "==> removing devbox-service=$label (disabled in config.yaml)"
    "${HF[@]}" -l "devbox-service=$label" destroy || true
  fi
}

"${HF[@]}" repos

sync_release postgres postgres
sync_release redis redis
sync_release temporal temporal
sync_release redpanda redpanda
sync_release observability observability

echo "Helmfile sync complete"
