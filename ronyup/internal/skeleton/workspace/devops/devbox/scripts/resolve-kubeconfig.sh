#!/usr/bin/env bash
# Print the kubeconfig path for the active cluster mode (no side effects).
# Invoked by: Makefile (KUBECONFIG ?= ...) and other tooling that needs the path only.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

resolve_kubeconfig "$ROOT"
