#!/usr/bin/env bash
# Register and start the vagrant-dns resolver for *.app.localdev (vagrant mode).
# Invoked by: bootstrap.sh, up.sh, resume.sh, and make dns.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

if [[ "$(cluster_mode "$ROOT")" != "vagrant" ]]; then
  exit 0
fi

vagrant dns --install
vagrant dns --start

echo "DNS ready: *.{$(dns_base "$ROOT")} via vagrant-dns"
