#!/usr/bin/env bash
# Block until kubectl can reach the cluster API (or timeout after ~5 minutes).
# Invoked by: cluster.sh and services.sh sync.
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

# Poll kubectl get nodes — works once the control plane is up and credentials are valid.
for i in $(seq 1 60); do
  if kubectl get nodes >/dev/null 2>&1; then
    echo "Kubernetes API is ready"
    exit 0
  fi
  sleep 5
done

echo "timed out waiting for Kubernetes API at $KUBECONFIG" >&2
exit 1
