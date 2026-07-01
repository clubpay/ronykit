#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

mode="$(cluster_mode "$ROOT")"

if [[ "$mode" == "vagrant" ]]; then
  vagrant up
  bash "$ROOT/scripts/kubeconfig.sh"
else
  export_kubeconfig_env "$ROOT"
  echo "Using cluster.mode=existing with KUBECONFIG=$KUBECONFIG"
fi

bash "$ROOT/scripts/wait-k8s.sh"
bash "$ROOT/scripts/install-services.sh"
