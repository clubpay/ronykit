#!/usr/bin/env bash
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

echo "==> Updating base packages"
apt-get update -y
apt-get upgrade -y
apt-get install -y \
  apt-transport-https \
  ca-certificates \
  curl \
  gnupg \
  jq \
  lsb-release \
  snapd

echo "==> Installing microk8s"
if ! snap list microk8s >/dev/null 2>&1; then
  snap install microk8s --classic --channel=1.32/stable
fi

usermod -aG microk8s vagrant
usermod -aG microk8s root

echo "==> Waiting for microk8s"
microk8s status --wait-ready

echo "==> Enabling microk8s addons"
microk8s enable dns storage ingress helm3

echo "==> Creating devbox namespace"
microk8s kubectl create namespace devbox --dry-run=client -o yaml | microk8s kubectl apply -f -

echo "==> Exporting kubeconfig to shared folder"
mkdir -p /vagrant/shared
microk8s config > /vagrant/shared/kubeconfig.raw

# Patch the API server address so the host can reach microk8s via the forwarded port.
sed 's|https://127.0.0.1:16443|https://127.0.0.1:16443|g' /vagrant/shared/kubeconfig.raw > /vagrant/shared/kubeconfig
chmod 0644 /vagrant/shared/kubeconfig

echo "==> Devbox VM provision complete"
