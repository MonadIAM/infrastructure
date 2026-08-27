#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"

. "$ROOT_DIR/scripts/lib/env.sh"
. "$ROOT_DIR/scripts/lib/helm.sh"

load_env
require_env \
    KUBE_CONTEXT \
    PROMETHEUS_COMMUNITY_HELM_REPO \
    KUBE_PROMETHEUS_STACK_CHART_VERSION \
    GRAFANA_COMMUNITY_HELM_REPO \
    OPENTELEMETRY_HELM_REPO \
    LOKI_CHART_VERSION \
    TEMPO_CHART_VERSION \
    OPENTELEMETRY_COLLECTOR_CHART_VERSION

cd "$ROOT_DIR"

crd_file="$(mktemp)"
trap 'rm -f "$crd_file"' EXIT

rollout_status() {
    kind="$1"
    name="$2"

    kubectl --context "$KUBE_CONTEXT" rollout status "$kind/$name" \
        -n monadiam-observability \
        --timeout=300s
}

wait_available() {
    kind="$1"
    name="$2"

    kubectl --context "$KUBE_CONTEXT" wait --for=condition=Available "$kind/$name" \
        -n monadiam-observability \
        --timeout=300s
}

kubectl --context "$KUBE_CONTEXT" apply -k clusters/local
kubectl --context "$KUBE_CONTEXT" wait --for=create secret/grafana-admin \
    -n monadiam-observability \
    --timeout=180s

helm_repo_add prometheus-community "$PROMETHEUS_COMMUNITY_HELM_REPO"
helm_repo_add grafana-community "$GRAFANA_COMMUNITY_HELM_REPO"
helm_repo_add open-telemetry "$OPENTELEMETRY_HELM_REPO"

helm_show_crds prometheus-community/kube-prometheus-stack \
    --version "$KUBE_PROMETHEUS_STACK_CHART_VERSION" > "$crd_file"
kubectl --context "$KUBE_CONTEXT" apply --server-side -f "$crd_file"

helm_upgrade_install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
    --version "$KUBE_PROMETHEUS_STACK_CHART_VERSION" \
    --namespace monadiam-observability \
    --create-namespace \
    --kube-context "$KUBE_CONTEXT" \
    --values infrastructure/observability/prometheus-stack/helm/values.yaml \
    --skip-crds \
    --timeout 600s

rollout_status deployment kube-prometheus-stack-operator
rollout_status deployment kube-prometheus-stack-grafana
rollout_status deployment kube-prometheus-stack-kube-state-metrics
rollout_status daemonset kube-prometheus-stack-prometheus-node-exporter
wait_available prometheus kube-prometheus-stack-prometheus
wait_available alertmanager kube-prometheus-stack-alertmanager

helm_upgrade_install loki grafana-community/loki \
    --version "$LOKI_CHART_VERSION" \
    --namespace monadiam-observability \
    --create-namespace \
    --kube-context "$KUBE_CONTEXT" \
    --values infrastructure/observability/loki/helm/values.yaml \
    --timeout 600s

rollout_status statefulset loki

helm_upgrade_install tempo grafana-community/tempo \
    --version "$TEMPO_CHART_VERSION" \
    --namespace monadiam-observability \
    --create-namespace \
    --kube-context "$KUBE_CONTEXT" \
    --values infrastructure/observability/tempo/helm/values.yaml \
    --timeout 600s

rollout_status statefulset tempo

helm_upgrade_install otel-collector open-telemetry/opentelemetry-collector \
    --version "$OPENTELEMETRY_COLLECTOR_CHART_VERSION" \
    --namespace monadiam-observability \
    --create-namespace \
    --kube-context "$KUBE_CONTEXT" \
    --values infrastructure/observability/otel-collector/helm/values.yaml \
    --timeout 600s

rollout_status deployment otel-collector

kubectl --context "$KUBE_CONTEXT" get pods,svc -n monadiam-observability
