#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

export_kubeconfig_env "$ROOT"

if [[ ! -f "$KUBECONFIG" ]]; then
  echo "kubeconfig not found at $KUBECONFIG" >&2
  exit 1
fi

for i in $(seq 1 60); do
  if kubectl get nodes >/dev/null 2>&1; then
    echo "Kubernetes API is ready"
    exit 0
  fi
  sleep 5
done

echo "timed out waiting for Kubernetes API at $KUBECONFIG" >&2
exit 1
