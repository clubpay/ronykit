#!/usr/bin/env bash
# Expose enabled devbox services: HTTP via a standard Ingress, raw TCP via
# Traefik IngressRouteTCP (or an nginx tcp-services configmap on clusters that
# still run nginx). Hostnames: <host>.<app.name>.<dns.tld> (see exposure.yaml).
# Invoked by: services.sh sync after Helm releases are synced.
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
    | yq -r '.items[] | select(.metadata.name == "traefik" or .metadata.name == "public" or .metadata.name == "nginx" or .metadata.name == "nginx-microk8s") | .metadata.name' \
    | head -n1)"
  if [[ -n "$class" && "$class" != "null" ]]; then
    echo "$class"
    return
  fi
  echo "traefik"
}

# traefik_available returns 0 when the Traefik IngressRouteTCP CRD is installed.
traefik_available() {
  kubectl get crd ingressroutetcps.traefik.io >/dev/null 2>&1
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
    kubectl delete ingress devbox-exposure-http -n "$NS" --ignore-not-found >/dev/null
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

# render_traefik_tcp emits one IngressRouteTCP per enabled tcp endpoint. The
# entrypoint name matches the endpoint id (defined in services/values/traefik.yaml);
# HostSNI(`*`) forwards all traffic on that dedicated port to the backend.
render_traefik_tcp() {
  local id service target_port

  for id in $(enabled_endpoints); do
    [[ "$(yq -r ".endpoints.${id}.protocol" "$EXPOSURE")" == "tcp" ]] || continue
    service="$(yq -r ".endpoints.${id}.service" "$EXPOSURE")"
    target_port="$(yq -r ".endpoints.${id}.targetPort" "$EXPOSURE")"
    cat <<EOF
---
apiVersion: traefik.io/v1alpha1
kind: IngressRouteTCP
metadata:
  name: devbox-tcp-${id}
  namespace: ${NS}
  labels:
    app.kubernetes.io/part-of: devbox
    app.kubernetes.io/component: exposure
spec:
  entryPoints:
    - ${id}
  routes:
    - match: HostSNI(\`*\`)
      services:
        - name: ${service}
          port: ${target_port}
EOF
  done
}

apply_traefik_tcp() {
  local docs
  # Clear stale routes first so toggled-off services are removed.
  kubectl delete ingressroutetcp -n "$NS" \
    -l app.kubernetes.io/component=exposure --ignore-not-found >/dev/null 2>&1 || true

  docs="$(render_traefik_tcp)"
  if [[ -n "$docs" ]]; then
    echo "$docs" | kubectl apply -f -
    echo "TCP exposure applied via Traefik IngressRouteTCP"
  else
    echo "no TCP services enabled"
  fi
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

http_docs="$(render_http_ingress "$CLASS")"
if [[ -n "$http_docs" ]]; then
  printf '%s\n' "$http_docs" | kubectl apply -f -
else
  echo "no HTTP services enabled"
fi

if traefik_available; then
  apply_traefik_tcp
else
  TCP_CM="$(ingress_tcp_configmap)"
  if [[ -n "$TCP_CM" ]]; then
    apply_tcp_to_ingress_controller "$TCP_CM"
    echo "TCP exposure applied via ${TCP_CM}"
  else
    echo "warning: no Traefik CRDs or nginx TCP configmap found; TCP services are cluster-internal only" >&2
    echo "         use vagrant mode (Traefik) or install an ingress controller with TCP support" >&2
  fi
fi

print_endpoints
