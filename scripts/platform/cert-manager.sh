#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"

. "$ROOT_DIR/scripts/lib/env.sh"
. "$ROOT_DIR/scripts/lib/helm.sh"

load_env
require_env KUBE_CONTEXT CERT_MANAGER_VERSION CERT_MANAGER_CHART

cd "$ROOT_DIR"

helm_upgrade_install cert-manager "$CERT_MANAGER_CHART" \
    --version "$CERT_MANAGER_VERSION" \
    --namespace cert-manager \
    --create-namespace \
    --kube-context "$KUBE_CONTEXT" \
    --values infrastructure/security/cert-manager/helm/values.yaml

kubectl --context "$KUBE_CONTEXT" rollout status deployment/cert-manager -n cert-manager --timeout=180s
kubectl --context "$KUBE_CONTEXT" rollout status deployment/cert-manager-cainjector -n cert-manager --timeout=180s
kubectl --context "$KUBE_CONTEXT" rollout status deployment/cert-manager-webhook -n cert-manager --timeout=180s

kubectl --context "$KUBE_CONTEXT" apply -k infrastructure/security/cert-manager/resources
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready clusterissuer/monadiam-selfsigned --timeout=60s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready certificate/monadiam-root-ca -n cert-manager --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready clusterissuer/monadiam-ca --timeout=60s

kubectl --context "$KUBE_CONTEXT" get clusterissuers
