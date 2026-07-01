#!/usr/bin/env bash
# Configure the host to resolve *.app.localdev via the devbox VM (vagrant mode).
# Uses dnsmasq on the VM (provision.sh) plus a macOS resolver file or /etc/hosts fallback.
# Invoked by: up.sh and resume.sh in vagrant mode.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

if [[ "$(cluster_mode "$ROOT")" != "vagrant" ]]; then
  exit 0
fi

BASE="$(dns_base "$ROOT")"
VM_IP="$(devbox_vm_ip "$ROOT")"
EXPOSURE="$ROOT/services/exposure.yaml"
RESOLVER_FILE="/etc/resolver/${BASE}"
HOSTS_MARKER="# devbox-${BASE}"

hosts_entries() {
  local id host fqdn
  for id in $(yq -r '.endpoints | keys | .[]' "$EXPOSURE"); do
    host="$(yq -r ".endpoints.${id}.host" "$EXPOSURE")"
    fqdn="$(endpoint_fqdn "$host" "$ROOT")"
    echo "${VM_IP} ${fqdn}"
  done
  echo "${VM_IP} ${BASE}"
}

install_macos_resolver() {
  if [[ "$(uname -s)" != "Darwin" ]]; then
    return 1
  fi
  if ! command -v sudo >/dev/null 2>&1; then
    return 1
  fi

  echo "==> configuring macOS resolver for *.${BASE} -> ${VM_IP}"
  sudo mkdir -p /etc/resolver
  printf 'nameserver %s\nport 53\n' "$VM_IP" | sudo tee "$RESOLVER_FILE" >/dev/null
  echo "resolver installed: ${RESOLVER_FILE}"
  return 0
}

install_hosts_fallback() {
  local tmp entries
  entries="$(hosts_entries)"
  tmp="$(mktemp)"

  if [[ -f /etc/hosts ]] && grep -qF "$HOSTS_MARKER" /etc/hosts; then
    awk -v marker="$HOSTS_MARKER" '
      $0 == marker { skip=1; next }
      skip && /^#/ { next }
      skip && NF==0 { skip=0; next }
      skip { next }
      { print }
    ' /etc/hosts > "$tmp"
  else
    cp /etc/hosts "$tmp"
  fi

  {
    echo "$HOSTS_MARKER"
    echo "$entries"
  } >> "$tmp"

  if command -v sudo >/dev/null 2>&1; then
    echo "==> updating /etc/hosts for ${BASE}"
    sudo cp "$tmp" /etc/hosts
  else
    echo "add these lines to /etc/hosts:" >&2
    echo "$HOSTS_MARKER" >&2
    echo "$entries" >&2
  fi
  rm -f "$tmp"
}

if ! wait_for_vagrant "$ROOT" 5; then
  echo "devbox VM is not running; DNS sync skipped" >&2
  exit 0
fi

if install_macos_resolver; then
  :
else
  install_hosts_fallback
fi

echo "DNS ready: db.${BASE}:5432, temporal.${BASE}:7233, … (see make services output)"
