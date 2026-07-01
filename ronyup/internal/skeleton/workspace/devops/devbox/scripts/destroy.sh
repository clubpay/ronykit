#!/usr/bin/env bash
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
