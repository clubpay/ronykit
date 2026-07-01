#!/usr/bin/env bash
# Verify host prerequisites for devbox (kubectl, helm, yq; + vagrant when cluster.mode=vagrant).
# Invoked by: make bootstrap (also runs as a dependency of make up).
# On success: ensures Helm chart repos exist and prepares shared/ for vagrant mode.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

# need exits non-zero when a required CLI tool is not on PATH.
need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required tool: $1" >&2
    return 1
  fi
}

mode="$(cluster_mode "$ROOT")"

# install_hint prints platform-specific install guidance when bootstrap fails.
install_hint() {
  cat >&2 <<EOF
Install the missing tools, then re-run: make bootstrap

  macOS (Homebrew):
    brew install kubectl helm yq

  Ubuntu/Debian:
    See https://kubernetes.io/docs/tasks/tools/
    curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    sudo snap install yq
EOF
  if [[ "$mode" == "vagrant" ]]; then
    cat >&2 <<EOF

For cluster.mode vagrant, also install Vagrant + a provider (VirtualBox, UTM, libvirt, …).
Default box: bento/ubuntu-24.04 (see config.yaml vm.box).
On Apple Silicon: export VAGRANT_DEFAULT_PROVIDER=utm
EOF
  fi
}

# Core tools required for every cluster mode.
missing=0
for tool in kubectl helm yq; do
  if ! need "$tool"; then
    missing=1
  fi
done

if [[ "$mode" == "vagrant" ]] && ! need vagrant; then
  missing=1
fi

if [[ "$missing" -ne 0 ]]; then
  install_hint
  exit 1
fi

# Register upstream chart repos so later helm installs do not fail.
helm_repo_ensure refresh

# Vagrant mode: optional guest-additions plugin and gitignored shared/ for kubeconfig.
if [[ "$mode" == "vagrant" ]]; then
  vagrant plugin install vagrant-vbguest 2>/dev/null || true
  vagrant plugin install vagrant-dns 2>/dev/null || true
  mkdir -p shared
  touch shared/.gitkeep
fi

chmod +x scripts/*.sh 2>/dev/null || true

echo "Devbox bootstrap OK (cluster.mode=$mode)"
