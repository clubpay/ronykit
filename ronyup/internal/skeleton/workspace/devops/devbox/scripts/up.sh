#!/usr/bin/env bash
# Bring devbox up: provision or connect to the cluster, then install enabled services.
# Invoked by: make up (after bootstrap).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

mode="$(cluster_mode "$ROOT")"

# Vagrant: start VM, wait for it, and refresh shared/kubeconfig for the host.
if [[ "$mode" == "vagrant" ]]; then
  vagrant up
  if ! wait_for_vagrant "$ROOT"; then
    echo "devbox VM did not reach running state after vagrant up" >&2
    exit 1
  fi
  bash "$ROOT/scripts/kubeconfig.sh"
else
  export_kubeconfig_env "$ROOT"
  echo "Using cluster.mode=existing with KUBECONFIG=$KUBECONFIG"
fi

# Wait for API, then sync Helm releases and optional raw manifests from config.yaml.
bash "$ROOT/scripts/wait-k8s.sh"
bash "$ROOT/scripts/install-services.sh"
bash "$ROOT/scripts/sync-dns.sh"
