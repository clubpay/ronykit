#!/usr/bin/env bash
# Register and start the vagrant-dns resolver for *.app.localdev (vagrant mode).
# Invoked by: bootstrap.sh, cluster.sh, and make dns.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

if [[ "$(cluster_mode "$ROOT")" != "vagrant" ]]; then
  exit 0
fi

stop_stale_vagrant_dns() {
  local pid command

  command -v lsof >/dev/null 2>&1 || return 0
  command -v ps >/dev/null 2>&1 || return 0

  while IFS= read -r pid; do
    [[ -n "$pid" ]] || continue
    command="$(ps -p "$pid" -o command= 2>/dev/null || true)"
    if [[ "$command" == *vagrant-dns* ]]; then
      echo "Stopping stale vagrant-dns process on 127.0.0.1:5300 (pid $pid)"
      kill "$pid" 2>/dev/null || true
    else
      echo "warning: UDP 127.0.0.1:5300 is used by another process: ${command:-pid $pid}" >&2
    fi
  done < <(lsof -nP -tiUDP:5300 2>/dev/null || true)
}

vagrant dns --install
stop_stale_vagrant_dns
vagrant dns --start

echo "DNS ready: *.{$(dns_base "$ROOT")} via vagrant-dns"
