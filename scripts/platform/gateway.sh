#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"

. "$ROOT_DIR/scripts/lib/env.sh"

load_env
require_env KUBE_CONTEXT GATEWAY_API_VERSION

cd "$ROOT_DIR"

kubectl --context "$KUBE_CONTEXT" apply -k clusters/local

kubectl --context "$KUBE_CONTEXT" apply --server-side=true \
    -f "https://github.com/kubernetes-sigs/gateway-api/releases/download/${GATEWAY_API_VERSION}/standard-install.yaml"

kubectl --context "$KUBE_CONTEXT" wait --for=condition=Established crd/gatewayclasses.gateway.networking.k8s.io --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Established crd/gateways.gateway.networking.k8s.io --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Established crd/httproutes.gateway.networking.k8s.io --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Established crd/referencegrants.gateway.networking.k8s.io --timeout=180s

kubectl --context "$KUBE_CONTEXT" rollout status deployment/istiod -n istio-system --timeout=180s

kubectl --context "$KUBE_CONTEXT" apply -k infrastructure/networking/gateway-api/resources
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready certificate/monadiam-localhost-gateway-tls -n istio-ingress --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Accepted gateway/public -n istio-ingress --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Programmed gateway/public -n istio-ingress --timeout=180s

kubectl --context "$KUBE_CONTEXT" get gateway public -n istio-ingress
kubectl --context "$KUBE_CONTEXT" get service -n istio-ingress
