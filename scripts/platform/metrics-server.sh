#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"

. "$ROOT_DIR/scripts/lib/env.sh"

load_env
require_env KUBE_CONTEXT

kubectl --context "$KUBE_CONTEXT" rollout status deployment/metrics-server -n kube-system --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Available apiservice/v1beta1.metrics.k8s.io --timeout=180s

attempt=0
until kubectl --context "$KUBE_CONTEXT" top nodes >/dev/null 2>&1 &&
        kubectl --context "$KUBE_CONTEXT" top pods -A >/dev/null 2>&1; do
    attempt=$((attempt + 1))
    if [ "$attempt" -ge 36 ]; then
        kubectl --context "$KUBE_CONTEXT" top nodes || true
        kubectl --context "$KUBE_CONTEXT" top pods -A || true
        exit 1
    fi
    sleep 5
done

kubectl --context "$KUBE_CONTEXT" top nodes
kubectl --context "$KUBE_CONTEXT" top pods -A
