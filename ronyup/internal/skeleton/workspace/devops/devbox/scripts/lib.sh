#!/usr/bin/env bash
# Shared helpers for devbox scripts. Source from other scripts after setting ROOT.
set -euo pipefail

cluster_mode() {
  local root="${1:?}"
  yq -r '.cluster.mode // "existing"' "$root/config.yaml"
}

resolve_kubeconfig() {
  local root="${1:?}"
  local mode cfg

  mode="$(cluster_mode "$root")"
  cfg="$(yq -r '.cluster.kubeconfig // ""' "$root/config.yaml")"

  if [[ -n "$cfg" && "$cfg" != "null" ]]; then
    if [[ "$cfg" != /* ]]; then
      cfg="$root/$cfg"
    fi
    echo "$cfg"
    return
  fi

  if [[ "$mode" == "vagrant" ]]; then
    echo "$root/shared/kubeconfig"
    return
  fi

  if [[ -n "${KUBECONFIG:-}" ]]; then
    # Use the first path when KUBECONFIG lists several (kubectl convention).
    echo "${KUBECONFIG%%:*}"
    return
  fi

  echo "${HOME}/.kube/config"
}

export_kubeconfig_env() {
  local root="${1:?}"
  export KUBECONFIG
  KUBECONFIG="$(resolve_kubeconfig "$root")"
  export KUBECONFIG
}
