#!/usr/bin/env bash
# Remove all devbox Helm releases.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/services"

helmfile -f helmfile.yaml destroy || true

echo "Helmfile releases removed"
