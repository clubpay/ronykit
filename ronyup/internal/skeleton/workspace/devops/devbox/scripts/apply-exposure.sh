#!/usr/bin/env bash
# Expose enabled devbox services via ingress (HTTP) and nginx TCP passthrough.
# Hostnames: <host>.<app.name>.<dns.tld> (see services/exposure.yaml).
# Invoked by: install-services.sh after Helm releases are synced.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib.sh
source "$ROOT/scripts/lib.sh"

EXPOSURE="$ROOT/services/exposure.yaml"
NS=devbox
BASE="$(dns_base "$ROOT")"

ingress_class() {
  local class
  class="$(kubectl get ingressclass -o json 2>/dev/null \
    | yq -r '.items[] | select(.metadata.name == "public" or .metadata.name == "nginx" or .metadata.name == "nginx-microk8s") | .metadata.name' \
    | head -n1)"
  if [[ -n "$class" && "$class" != "null" ]]; then
    echo "$class"
    return
  fi
  echo "public"
}

ingress_tcp_configmap() {
  if kubectl get configmap nginx-ingress-tcp-microk8s-conf -n ingress >/dev/null 2>&1; then
    echo "ingress/nginx-ingress-tcp-microk8s-conf"
    return
  fi
  if kubectl get configmap tcp-services -n ingress-nginx >/dev/null 2>&1; then
    echo "ingress-nginx/tcp-services"
    return
  fi
  echo ""
}

enabled_endpoints() {
  local id key
  for id in $(yq -r '.endpoints | keys | .[]' "$EXPOSURE"); do
    key="$(yq -r ".endpoints.${id}.config_key" "$EXPOSURE")"
    if service_enabled "$ROOT" "$key"; then
      echo "$id"
    fi
  done
}

render_http_ingress() {
  local class="$1"
  local rules=""
  local id host service target_port

  for id in $(enabled_endpoints); do
    [[ "$(yq -r ".endpoints.${id}.protocol" "$EXPOSURE")" == "http" ]] || continue
    host="$(yq -r ".endpoints.${id}.host" "$EXPOSURE")"
    service="$(yq -r ".endpoints.${id}.service" "$EXPOSURE")"
    target_port="$(yq -r ".endpoints.${id}.targetPort" "$EXPOSURE")"
    rules="${rules}
  - host: ${host}.${BASE}
    http:
      paths:
        - path: /
          pathType: Prefix
          backend:
            service:
              name: ${service}
              port:
                number: ${target_port}"
  done

  if [[ -z "$rules" ]]; then
    kubectl delete ingress devbox-exposure-http -n "$NS" --ignore-not-found
    return 0
  fi

  cat <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: devbox-exposure-http
  namespace: ${NS}
  labels:
    app.kubernetes.io/part-of: devbox
    app.kubernetes.io/component: exposure
spec:
  ingressClassName: ${class}
  rules:${rules}
EOF
}

apply_tcp_to_ingress_controller() {
  local cm_ref="$1"
  local cm_ns="${cm_ref%%/*}"
  local cm_name="${cm_ref##*/}"
  local -a literals=()
  local id port service target_port

  for id in $(enabled_endpoints); do
    [[ "$(yq -r ".endpoints.${id}.protocol" "$EXPOSURE")" == "tcp" ]] || continue
    port="$(yq -r ".endpoints.${id}.port" "$EXPOSURE")"
    service="$(yq -r ".endpoints.${id}.service" "$EXPOSURE")"
    target_port="$(yq -r ".endpoints.${id}.targetPort" "$EXPOSURE")"
    literals+=("--from-literal=${port}=${NS}/${service}:${target_port}")
  done

  if [[ ${#literals[@]} -eq 0 ]]; then
    kubectl patch configmap "$cm_name" -n "$cm_ns" --type merge -p '{"data":{}}' >/dev/null 2>&1 || true
    return 0
  fi

  kubectl create configmap "$cm_name" -n "$cm_ns" "${literals[@]}" --dry-run=client -o yaml | kubectl apply -f -
}

print_endpoints() {
  local id host port protocol fqdn
  echo ""
  echo "Devbox endpoints (${BASE}):"
  for id in $(enabled_endpoints); do
    host="$(yq -r ".endpoints.${id}.host" "$EXPOSURE")"
    port="$(yq -r ".endpoints.${id}.port" "$EXPOSURE")"
    protocol="$(yq -r ".endpoints.${id}.protocol" "$EXPOSURE")"
    fqdn="$(endpoint_fqdn "$host" "$ROOT")"
    if [[ "$protocol" == "http" ]]; then
      echo "  http://${fqdn}/  (${id})"
    else
      echo "  ${fqdn}:${port}  (${id})"
    fi
  done
  echo ""
}

export_kubeconfig_env "$ROOT"

CLASS="$(ingress_class)"
TCP_CM="$(ingress_tcp_configmap)"

render_http_ingress "$CLASS" | kubectl apply -f -

if [[ -n "$TCP_CM" ]]; then
  apply_tcp_to_ingress_controller "$TCP_CM"
  echo "TCP exposure applied via ${TCP_CM}"
else
  echo "warning: no nginx ingress TCP configmap found; TCP services are cluster-internal only" >&2
  echo "         enable microk8s ingress (vagrant mode) or install nginx ingress with tcp-services" >&2
fi

print_endpoints
