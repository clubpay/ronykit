#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required tool: $1" >&2
    return 1
  fi
}

mode="$(cluster_mode "$ROOT")"

install_hint() {
  cat >&2 <<EOF
Install the missing tools, then re-run: make bootstrap

  macOS (Homebrew):
    brew install kubectl helm helmfile yq

  Ubuntu/Debian:
    See https://kubernetes.io/docs/tasks/tools/
    curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    curl -fsSL https://github.com/helmfile/helmfile/releases/latest/download/helmfile_linux_amd64.tar.gz | tar xz -C /usr/local/bin helmfile
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

missing=0
for tool in kubectl helm helmfile yq; do
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

if [[ "$mode" == "vagrant" ]]; then
  vagrant plugin install vagrant-vbguest 2>/dev/null || true
  mkdir -p shared
  touch shared/.gitkeep
fi

chmod +x scripts/*.sh 2>/dev/null || true

echo "Devbox bootstrap OK (cluster.mode=$mode)"
