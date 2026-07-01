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
  snap install microk8s --classic --channel=1.32/stable
fi

usermod -aG microk8s vagrant
usermod -aG microk8s root

echo "==> Waiting for microk8s"
microk8s status --wait-ready

echo "==> Enabling microk8s addons (one at a time)"
microk8s enable dns
microk8s enable hostpath-storage
microk8s enable ingress
microk8s enable helm3

APP_NAME="${DEVBOX_APP_NAME:-app}"
DNS_TLD="${DEVBOX_DNS_TLD:-localdev}"
VM_IP="${DEVBOX_VM_IP:-192.168.56.10}"

echo "==> Configuring dnsmasq for *.${APP_NAME}.${DNS_TLD}"
apt-get install -y dnsmasq
cat > /etc/dnsmasq.d/devbox.conf <<EOF
# Wildcard DNS: any <service>.${APP_NAME}.${DNS_TLD} resolves to the devbox VM.
listen-address=127.0.0.1,${VM_IP}
bind-interfaces
address=/.${APP_NAME}.${DNS_TLD}/${VM_IP}
EOF
systemctl enable dnsmasq
systemctl restart dnsmasq

echo "==> Creating devbox namespace"
microk8s kubectl create namespace devbox --dry-run=client -o yaml | microk8s kubectl apply -f -

echo "==> Exporting kubeconfig to shared folder"
mkdir -p /vagrant/shared
microk8s config > /vagrant/shared/kubeconfig.raw

# Host kubectl uses the forwarded port; keep the server URL consistent with kubeconfig.sh.
sed 's|https://127.0.0.1:16443|https://127.0.0.1:16443|g' /vagrant/shared/kubeconfig.raw > /vagrant/shared/kubeconfig
chmod 0644 /vagrant/shared/kubeconfig

echo "==> Devbox VM provision complete"
