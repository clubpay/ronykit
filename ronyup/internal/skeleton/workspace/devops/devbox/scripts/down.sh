#!/usr/bin/env bash
# Tear down devbox without deleting the cluster.
# Invoked by: make down
#   vagrant mode  → halt VM (services stay on disk inside the VM)
#   existing mode → uninstall Helm releases and optional tigerbeetle manifest
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

kubectl delete ingress devbox-exposure-http -n devbox --ignore-not-found
if kubectl get configmap nginx-ingress-tcp-microk8s-conf -n ingress >/dev/null 2>&1; then
  kubectl patch configmap nginx-ingress-tcp-microk8s-conf -n ingress --type merge -p '{"data":{}}' >/dev/null 2>&1 || true
fi
if kubectl get configmap tcp-services -n ingress-nginx >/dev/null 2>&1; then
  kubectl patch configmap tcp-services -n ingress-nginx --type merge -p '{"data":{}}' >/dev/null 2>&1 || true
fi

# Tigerbeetle is applied outside Helm; remove it if it was enabled.
if yq -e '.services.tigerbeetle == true' "$ROOT/config.yaml" >/dev/null 2>&1; then
  kubectl delete -f "$ROOT/services/manifests/tigerbeetle.yaml" --ignore-not-found
fi

echo "Devbox services removed from cluster"
