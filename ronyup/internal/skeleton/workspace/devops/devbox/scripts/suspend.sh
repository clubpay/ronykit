#!/usr/bin/env bash
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
