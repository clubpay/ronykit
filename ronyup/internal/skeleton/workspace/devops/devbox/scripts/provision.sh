#!/usr/bin/env bash
# First-boot provisioning for the devbox Vagrant VM (microk8s + devbox namespace).
# Invoked by: Vagrantfile provisioner (not called directly from the Makefile).
# Writes admin kubeconfig into /vagrant/shared for the host via synced folder.
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
  snap install microk8s --classic --channel=1.35/stable
fi

usermod -aG microk8s vagrant
usermod -aG microk8s root

echo "==> Waiting for microk8s"
microk8s status --wait-ready

echo "==> Enabling microk8s addons (one at a time)"
microk8s enable dns
microk8s enable hostpath-storage
microk8s enable helm3

echo "==> Installing Traefik ingress (Helm)"
# Traefik replaces the microk8s nginx ingress addon: it handles HTTP ingress and
# raw TCP passthrough (IngressRouteTCP) for the database ports. Values define the
# entrypoints; see services/values/traefik.yaml (synced at /vagrant).
microk8s helm3 repo add traefik https://traefik.github.io/charts
microk8s helm3 repo update
microk8s helm3 upgrade --install traefik traefik/traefik \
  --namespace traefik \
  --create-namespace \
  -f /vagrant/services/values/traefik.yaml \
  --wait \
  --timeout 15m

echo "==> Creating devbox namespace"
microk8s kubectl create namespace devbox --dry-run=client -o yaml | microk8s kubectl apply -f -

echo "==> Exporting kubeconfig to shared folder"
mkdir -p /vagrant/shared
microk8s config > /vagrant/shared/kubeconfig.raw

# Host kubectl uses the forwarded port; keep the server URL consistent with kubeconfig.sh.
sed 's|https://127.0.0.1:16443|https://127.0.0.1:16443|g' /vagrant/shared/kubeconfig.raw > /vagrant/shared/kubeconfig
chmod 0644 /vagrant/shared/kubeconfig

echo "==> Devbox VM provision complete"
