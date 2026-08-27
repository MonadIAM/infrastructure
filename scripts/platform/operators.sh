#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"

. "$ROOT_DIR/scripts/lib/env.sh"

load_env
require_env \
    KUBE_CONTEXT \
    CNPG_HELM_REPO \
    CNPG_CHART_VERSION \
    STRIMZI_HELM_REPO \
    STRIMZI_CHART_VERSION \
    ALTINITY_HELM_REPO \
    CLICKHOUSE_OPERATOR_CHART_VERSION \
    MINIO_OPERATOR_HELM_REPO \
    MINIO_OPERATOR_CHART_VERSION

cd "$ROOT_DIR"

sync_minio_operator_ca() {
    ca_crt="$(kubectl --context "$KUBE_CONTEXT" get secret monadiam-root-ca -n cert-manager -o jsonpath='{.data.ca\.crt}')"
    if [ -z "$ca_crt" ]; then
        echo "cert-manager root CA secret cert-manager/monadiam-root-ca does not contain ca.crt" >&2
        exit 1
    fi

    kubectl --context "$KUBE_CONTEXT" apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: operator-ca-tls
  namespace: minio-operator
  labels:
    app.kubernetes.io/name: operator-ca-tls
    app.kubernetes.io/component: object-storage-operator
    app.kubernetes.io/part-of: monadiam
    app.kubernetes.io/managed-by: scripts
type: Opaque
data:
  public.crt: $ca_crt
EOF
}

kubectl --context "$KUBE_CONTEXT" apply -k clusters/local

helm repo add cnpg "$CNPG_HELM_REPO" --force-update
helm repo add strimzi "$STRIMZI_HELM_REPO" --force-update
helm repo add altinity "$ALTINITY_HELM_REPO" --force-update
helm repo add minio "$MINIO_OPERATOR_HELM_REPO" --force-update
helm repo update cnpg strimzi altinity minio

helm upgrade --install cnpg cnpg/cloudnative-pg \
    --version "$CNPG_CHART_VERSION" \
    --namespace cnpg-system \
    --create-namespace \
    --kube-context "$KUBE_CONTEXT" \
    --values infrastructure/operators/cnpg/helm/values.yaml

kubectl --context "$KUBE_CONTEXT" rollout status deployment/cnpg-cloudnative-pg -n cnpg-system --timeout=600s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Established crd/clusters.postgresql.cnpg.io --timeout=600s

helm upgrade --install strimzi strimzi/strimzi-kafka-operator \
    --version "$STRIMZI_CHART_VERSION" \
    --namespace strimzi-system \
    --create-namespace \
    --kube-context "$KUBE_CONTEXT" \
    --values infrastructure/operators/strimzi/helm/values.yaml

kubectl --context "$KUBE_CONTEXT" rollout status deployment/strimzi-cluster-operator -n strimzi-system --timeout=600s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Established crd/kafkas.kafka.strimzi.io --timeout=600s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Established crd/kafkanodepools.kafka.strimzi.io --timeout=600s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Established crd/kafkaconnects.kafka.strimzi.io --timeout=600s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Established crd/kafkausers.kafka.strimzi.io --timeout=600s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Established crd/kafkatopics.kafka.strimzi.io --timeout=600s

helm upgrade --install clickhouse-operator altinity/altinity-clickhouse-operator \
    --version "$CLICKHOUSE_OPERATOR_CHART_VERSION" \
    --namespace clickhouse-operator \
    --create-namespace \
    --kube-context "$KUBE_CONTEXT" \
    --values infrastructure/operators/clickhouse/helm/values.yaml

kubectl --context "$KUBE_CONTEXT" rollout restart deployment/clickhouse-operator-altinity-clickhouse-operator -n clickhouse-operator
kubectl --context "$KUBE_CONTEXT" rollout status deployment/clickhouse-operator-altinity-clickhouse-operator -n clickhouse-operator --timeout=600s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Established crd/clickhouseinstallations.clickhouse.altinity.com --timeout=600s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Established crd/clickhousekeeperinstallations.clickhouse-keeper.altinity.com --timeout=600s

sync_minio_operator_ca
helm upgrade --install minio-operator minio/operator \
    --version "$MINIO_OPERATOR_CHART_VERSION" \
    --namespace minio-operator \
    --create-namespace \
    --kube-context "$KUBE_CONTEXT" \
    --values infrastructure/operators/minio/helm/values.yaml

kubectl --context "$KUBE_CONTEXT" rollout restart deployment/minio-operator -n minio-operator
kubectl --context "$KUBE_CONTEXT" rollout status deployment/minio-operator -n minio-operator --timeout=600s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Established crd/tenants.minio.min.io --timeout=600s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Established crd/policybindings.sts.min.io --timeout=600s

kubectl --context "$KUBE_CONTEXT" get deployments -n cnpg-system
kubectl --context "$KUBE_CONTEXT" get deployments -n strimzi-system
kubectl --context "$KUBE_CONTEXT" get deployments -n clickhouse-operator
kubectl --context "$KUBE_CONTEXT" get deployments -n minio-operator
kubectl --context "$KUBE_CONTEXT" get crd clusters.postgresql.cnpg.io kafkas.kafka.strimzi.io kafkaconnects.kafka.strimzi.io clickhouseinstallations.clickhouse.altinity.com clickhousekeeperinstallations.clickhouse-keeper.altinity.com tenants.minio.min.io policybindings.sts.min.io
