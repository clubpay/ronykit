#!/usr/bin/env bash
# Refresh shared/kubeconfig so the host can reach microk8s inside the Vagrant VM.
# Invoked by: make kubeconfig (also called from up.sh and resume.sh in vagrant mode).
# Rewrites the API server URL to https://127.0.0.1:16443 (forwarded port on the host).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

if [[ "$(cluster_mode "$ROOT")" != "vagrant" ]]; then
  echo "kubeconfig refresh applies only when cluster.mode is vagrant" >&2
  exit 1
fi

mkdir -p "$ROOT/shared"

# Fallback when the synced folder is empty: SSH into the VM and dump microk8s config.
fetch_kubeconfig_from_vm() {
  vagrant ssh -c "microk8s config" > "$ROOT/shared/kubeconfig.raw"
}

# Provision writes kubeconfig into the synced folder; give the host a moment to see it.
wait_for_kubeconfig_file() {
  local attempts="${1:-60}"
  local i

  for ((i = 1; i <= attempts; i++)); do
    if [[ -f "$ROOT/shared/kubeconfig.raw" || -f "$ROOT/shared/kubeconfig" ]]; then
      return 0
    fi
    sleep 2
  done

  return 1
}

if ! wait_for_kubeconfig_file 5; then
  if ! wait_for_vagrant "$ROOT"; then
    echo "devbox VM did not reach running state" >&2
    exit 1
  fi

  if ! fetch_kubeconfig_from_vm; then
    if ! wait_for_kubeconfig_file 30; then
      echo "could not obtain kubeconfig from the VM or shared folder" >&2
      exit 1
    fi
  fi
fi

if [[ -f "$ROOT/shared/kubeconfig.raw" ]]; then
  cp "$ROOT/shared/kubeconfig.raw" "$ROOT/shared/kubeconfig"
elif [[ ! -f "$ROOT/shared/kubeconfig" ]]; then
  echo "kubeconfig file missing after sync" >&2
  exit 1
fi

# Normalize the API endpoint for host-side kubectl (Vagrant forwards 16443 → VM).
if grep -q 'server: https://' "$ROOT/shared/kubeconfig"; then
  sed -i.bak 's|server: https://[^ ]*|server: https://127.0.0.1:16443|' "$ROOT/shared/kubeconfig"
  rm -f "$ROOT/shared/kubeconfig.bak"
fi

chmod 0644 "$ROOT/shared/kubeconfig"
echo "kubeconfig written to $ROOT/shared/kubeconfig"
echo "export KUBECONFIG=$ROOT/shared/kubeconfig"
