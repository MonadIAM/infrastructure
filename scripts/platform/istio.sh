#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"

. "$ROOT_DIR/scripts/lib/env.sh"
. "$ROOT_DIR/scripts/lib/helm.sh"

load_env
require_env KUBE_CONTEXT ISTIO_VERSION ISTIO_HELM_REPO ISTIO_PLATFORM

cd "$ROOT_DIR"

kubectl --context "$KUBE_CONTEXT" apply -k clusters/local

helm_repo_add istio "$ISTIO_HELM_REPO"

helm_upgrade_install istio-base istio/base \
    --version "$ISTIO_VERSION" \
    --namespace istio-system \
    --create-namespace \
    --kube-context "$KUBE_CONTEXT" \
    --set "platform=$ISTIO_PLATFORM" \
    --values infrastructure/networking/istio/helm/base-values.yaml

helm_upgrade_install istiod istio/istiod \
    --version "$ISTIO_VERSION" \
    --namespace istio-system \
    --kube-context "$KUBE_CONTEXT" \
    --set "platform=$ISTIO_PLATFORM" \
    --values infrastructure/networking/istio/helm/istiod-values.yaml

kubectl --context "$KUBE_CONTEXT" rollout status deployment/istiod -n istio-system --timeout=180s
kubectl --context "$KUBE_CONTEXT" get deployment istiod -n istio-system
