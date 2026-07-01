#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

mode="$(cluster_mode "$ROOT")"

if [[ "$mode" == "vagrant" ]]; then
  vagrant halt
  exit 0
fi

export_kubeconfig_env "$ROOT"
cd "$ROOT"

echo "Removing devbox Helm releases from the current cluster (cluster.mode=existing)"
bash "$ROOT/scripts/helmfile-destroy.sh"

if yq -e '.services.tigerbeetle == true' "$ROOT/config.yaml" >/dev/null 2>&1; then
  kubectl delete -f "$ROOT/services/manifests/tigerbeetle.yaml" --ignore-not-found
fi

echo "Devbox services removed from cluster"
