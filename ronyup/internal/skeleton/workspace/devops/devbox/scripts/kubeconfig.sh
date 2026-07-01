#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

if [[ "$(cluster_mode "$ROOT")" != "vagrant" ]]; then
  echo "kubeconfig refresh applies only when cluster.mode is vagrant" >&2
  exit 1
fi

mkdir -p "$ROOT/shared"

if vagrant status --machine-readable 2>/dev/null | grep -q ',state,running,'; then
  vagrant ssh -c "microk8s config" > "$ROOT/shared/kubeconfig.raw"
else
  if [[ ! -f "$ROOT/shared/kubeconfig.raw" ]]; then
    echo "devbox VM is not running and no cached kubeconfig exists" >&2
    exit 1
  fi
fi

if [[ -f "$ROOT/shared/kubeconfig.raw" ]]; then
  cp "$ROOT/shared/kubeconfig.raw" "$ROOT/shared/kubeconfig"
fi

if grep -q 'server: https://' "$ROOT/shared/kubeconfig"; then
  sed -i.bak 's|server: https://[^ ]*|server: https://127.0.0.1:16443|' "$ROOT/shared/kubeconfig"
  rm -f "$ROOT/shared/kubeconfig.bak"
fi

chmod 0644 "$ROOT/shared/kubeconfig"
echo "kubeconfig written to $ROOT/shared/kubeconfig"
echo "export KUBECONFIG=$ROOT/shared/kubeconfig"
