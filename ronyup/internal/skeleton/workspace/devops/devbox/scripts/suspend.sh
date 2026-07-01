#!/usr/bin/env bash
# Suspend the devbox Vagrant VM, preserving disk and memory state.
# Invoked by: make suspend (alias: make pause). No-op for cluster.mode=existing.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

if [[ "$(cluster_mode "$ROOT")" != "vagrant" ]]; then
  echo "suspend applies only when cluster.mode is vagrant" >&2
  exit 1
fi

vagrant suspend
echo "Devbox VM suspended (use: make resume)"
