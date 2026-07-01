#!/usr/bin/env bash
# Fully remove devbox infrastructure.
# Invoked by: make destroy
#   vagrant mode  → destroy VM and all its data (vagrant destroy -f)
#   existing mode → same as make down (uninstall releases from the shared cluster)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

mode="$(cluster_mode "$ROOT")"

if [[ "$mode" == "vagrant" ]]; then
  vagrant destroy -f
  exit 0
fi

bash "$ROOT/scripts/down.sh"
