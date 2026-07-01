#!/usr/bin/env bash
# Shared helpers for devbox scripts. Source from other scripts after setting ROOT.
set -euo pipefail

# cluster_mode reads cluster.mode from config.yaml (default: existing).
cluster_mode() {
  local root="${1:?}"
  yq -r '.cluster.mode // "existing"' "$root/config.yaml"
}

# resolve_kubeconfig picks the kubeconfig path for the active cluster mode:
#   config.yaml cluster.kubeconfig → vagrant shared/kubeconfig → KUBECONFIG → ~/.kube/config
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

# export_kubeconfig_env sets KUBECONFIG for kubectl/helm subprocesses.
export_kubeconfig_env() {
  local root="${1:?}"
  export KUBECONFIG
  KUBECONFIG="$(resolve_kubeconfig "$root")"
  export KUBECONFIG
}

# vagrant_running returns 0 when the devbox VM reports state=running.
vagrant_running() {
  vagrant status --machine-readable 2>/dev/null | grep -qE ',state,running$'
}

# wait_for_vagrant blocks until the VM reports running or times out.
wait_for_vagrant() {
  local root="${1:?}"
  local attempts="${2:-60}"
  local i

  cd "$root"

  for ((i = 1; i <= attempts; i++)); do
    if vagrant_running; then
      return 0
    fi
    sleep 2
  done

  return 1
}

# helm_repo_ensure adds missing Helm repos and runs helm repo update.
helm_repo_ensure() {
  local name url

  while IFS='|' read -r name url; do
    [[ -z "$name" ]] && continue
    if ! helm repo list 2>/dev/null | awk -v n="$name" '$1 == n { found = 1 } END { exit !found }'; then
      helm repo add "$name" "$url"
    fi
  done <<'EOF'
bitnami|https://charts.bitnami.com/bitnami
temporalio|https://go.temporal.io/helm-charts
redpanda|https://charts.redpanda.com
open-telemetry|https://open-telemetry.github.io/opentelemetry-helm-charts
jaegertracing|https://jaegertracing.github.io/helm-charts
grafana|https://grafana.github.io/helm-charts
EOF

  helm repo update
}
