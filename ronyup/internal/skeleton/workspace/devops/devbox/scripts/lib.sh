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

# helm_repo_exists returns 0 when the named Helm repo is already registered.
helm_repo_exists() {
  local name="${1:?}"
  helm repo list -o json 2>/dev/null \
    | yq -e ".[] | select(.name == \"$name\")" >/dev/null 2>&1
}

# helm_repo_ensure registers missing chart repos.
# Pass "refresh" to run `helm repo update` (bootstrap); otherwise only adds missing repos.
helm_repo_ensure() {
  local refresh="${1:-}"
  local name url added=0

  while IFS='|' read -r name url; do
    [[ -z "$name" ]] && continue
    if helm_repo_exists "$name"; then
      continue
    fi
    echo "==> adding helm repo: $name"
    helm repo add "$name" "$url"
    added=1
  done <<'EOF'
cnpg|https://cloudnative-pg.github.io/charts
temporalio|https://go.temporal.io/helm-charts
redpanda|https://charts.redpanda.com
open-telemetry|https://open-telemetry.github.io/opentelemetry-helm-charts
jaegertracing|https://jaegertracing.github.io/helm-charts
grafana|https://grafana.github.io/helm-charts
EOF

  if [[ "$refresh" == "refresh" || "$added" -eq 1 ]]; then
    helm repo update
  fi
}

# app_name reads app.name from config.yaml (scaffold default: app).
app_name() {
  local root="${1:?}"
  yq -r '.app.name // "app"' "$root/config.yaml"
}

# dns_tld reads dns.tld from config.yaml (default: localdev).
dns_tld() {
  local root="${1:?}"
  yq -r '.dns.tld // "localdev"' "$root/config.yaml"
}

# dns_base returns <app.name>.<dns.tld> (e.g. myapp.localdev).
dns_base() {
  local root="${1:?}"
  echo "$(app_name "$root").$(dns_tld "$root")"
}

# endpoint_fqdn returns <host>.<dns.base> (e.g. db.myapp.localdev).
endpoint_fqdn() {
  local host="${1:?}"
  local root="${2:?}"
  echo "${host}.$(dns_base "$root")"
}

# service_enabled checks a boolean toggle under services.* in config.yaml.
service_enabled() {
  local root="${1:?}"
  local key="${2:?}"
  yq -e ".services.${key} == true" "$root/config.yaml" >/dev/null 2>&1
}

# require_vagrant_mode exits unless cluster.mode is vagrant.
require_vagrant_mode() {
  local root="${1:?}"
  local action="${2:?}"

  if [[ "$(cluster_mode "$root")" != "vagrant" ]]; then
    echo "${action} applies only when cluster.mode is vagrant (see config.yaml)" >&2
    exit 1
  fi
}

# devbox_script runs another script from the devbox scripts/ directory.
devbox_script() {
  local root="${1:?}"
  local name="${2:?}"
  shift 2

  bash "$root/scripts/$name" "$@"
}
