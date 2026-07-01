#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

export_kubeconfig_env "$ROOT"

bash "$ROOT/scripts/wait-k8s.sh"

kubectl create namespace devbox --dry-run=client -o yaml | kubectl apply -f -

bash "$ROOT/scripts/helmfile-apply.sh"

if yq -e '.services.tigerbeetle == true' "$ROOT/config.yaml" >/dev/null 2>&1; then
  kubectl apply -f "$ROOT/services/manifests/tigerbeetle.yaml"
else
  kubectl delete -f "$ROOT/services/manifests/tigerbeetle.yaml" --ignore-not-found
fi

echo "Devbox services applied (see config.yaml for enabled set)"
