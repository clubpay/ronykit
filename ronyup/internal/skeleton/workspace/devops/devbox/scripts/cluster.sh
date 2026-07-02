#!/usr/bin/env bash
# Devbox cluster / VM lifecycle.
#
# Usage: cluster.sh <command>
#   up       - start or connect to the cluster, then sync enabled services
#   down     - halt VM (vagrant) or remove devbox services (existing cluster)
#   suspend  - suspend VM, preserving state (vagrant only; alias: pause)
#   resume   - resume a suspended VM (vagrant only)
#   destroy  - destroy VM and data (vagrant) or remove services (existing cluster)
#
# Invoked by: make up | down | suspend | pause | resume | destroy
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

usage() {
  cat <<EOF
Usage: $(basename "$0") <command>

Commands:
  up        start or connect to the cluster and sync enabled services
  down      halt VM (vagrant) or remove devbox services (existing cluster)
  suspend   suspend VM, saving its state (vagrant only; alias: pause)
  resume    resume a suspended VM (vagrant only)
  destroy   destroy VM and data (vagrant) or remove services (existing cluster)
  help      show this help

Edit config.yaml to choose cluster.mode (existing|vagrant) and service toggles.
EOF
}

cmd_up() {
  local mode
  mode="$(cluster_mode "$ROOT")"

  if [[ "$mode" == "vagrant" ]]; then
    vagrant up
    if ! wait_for_vagrant "$ROOT"; then
      echo "devbox VM did not reach running state after vagrant up" >&2
      exit 1
    fi
    devbox_script "$ROOT" kubeconfig.sh
  else
    export_kubeconfig_env "$ROOT"
    echo "Using cluster.mode=existing with KUBECONFIG=$KUBECONFIG"
  fi

  devbox_script "$ROOT" wait-k8s.sh
  devbox_script "$ROOT" dns.sh
  devbox_script "$ROOT" services.sh sync
}

cmd_down() {
  local mode
  mode="$(cluster_mode "$ROOT")"

  if [[ "$mode" == "vagrant" ]]; then
    vagrant halt
    echo "Devbox VM halted (use: make up or make resume)"
    return 0
  fi

  echo "Removing devbox services from the current cluster (cluster.mode=existing)"
  devbox_script "$ROOT" services.sh remove
}

cmd_suspend() {
  require_vagrant_mode "$ROOT" "suspend"
  vagrant suspend
  echo "Devbox VM suspended (use: make resume)"
}

cmd_resume() {
  require_vagrant_mode "$ROOT" "resume"
  vagrant resume
  devbox_script "$ROOT" kubeconfig.sh
  devbox_script "$ROOT" wait-k8s.sh
  devbox_script "$ROOT" dns.sh
  echo "Devbox VM resumed"
}

cmd_destroy() {
  local mode
  mode="$(cluster_mode "$ROOT")"

  if [[ "$mode" == "vagrant" ]]; then
    vagrant destroy -f
    echo "Devbox VM destroyed"
    return 0
  fi

  cmd_down
}

main() {
  local cmd="${1:-}"

  case "$cmd" in
    up)
      cmd_up
      ;;
    down)
      cmd_down
      ;;
    suspend | pause)
      cmd_suspend
      ;;
    resume)
      cmd_resume
      ;;
    destroy)
      cmd_destroy
      ;;
    help | -h | --help)
      usage
      ;;
    "")
      echo "missing command" >&2
      usage >&2
      exit 1
      ;;
    *)
      echo "unknown command: $cmd" >&2
      usage >&2
      exit 1
      ;;
  esac
}

main "$@"
