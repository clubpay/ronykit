#!/usr/bin/env bash
# Resume a suspended devbox VM and wait for Kubernetes to become reachable again.
# Invoked by: make resume. No-op for cluster.mode=existing.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

if [[ "$(cluster_mode "$ROOT")" != "vagrant" ]]; then
  echo "resume applies only when cluster.mode is vagrant" >&2
  exit 1
fi

vagrant resume
# Kubeconfig and API endpoint may need refresh after suspend/resume.
bash "$ROOT/scripts/kubeconfig.sh"
bash "$ROOT/scripts/wait-k8s.sh"
bash "$ROOT/scripts/sync-dns.sh"
echo "Devbox VM resumed"
