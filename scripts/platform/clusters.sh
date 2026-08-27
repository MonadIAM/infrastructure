#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"

. "$ROOT_DIR/scripts/lib/env.sh"
. "$ROOT_DIR/scripts/lib/helm.sh"
. "$ROOT_DIR/scripts/lib/minio.sh"

load_env
require_env \
    KUBE_CONTEXT \
    REDIS_CHART \
    REDIS_CHART_VERSION

wait_clickhouse_installation_completed() {
    namespace="$1"
    name="$2"
    attempt=0

    while :; do
        json="$(kubectl --context "$KUBE_CONTEXT" get clickhouseinstallations.clickhouse.altinity.com "$name" -n "$namespace" -o json 2>/dev/null || true)"
        if [ -n "$json" ] && printf '%s\n' "$json" | jq -e . >/dev/null 2>&1; then
            status="$(printf '%s\n' "$json" | jq -r '.status.status // ""')"
            if [ "$status" = "Completed" ]; then
                echo "ClickHouseInstallation $namespace/$name completed"
                return
            fi
        fi

        attempt=$((attempt + 1))
        if [ "$attempt" -ge 120 ]; then
            kubectl --context "$KUBE_CONTEXT" describe clickhouseinstallations.clickhouse.altinity.com "$name" -n "$namespace" || true
            exit 1
        fi
        sleep 5
    done
}

wait_minio_tenant_initialized() {
    namespace="$1"
    name="$2"
    attempt=0

    while :; do
        json="$(kubectl --context "$KUBE_CONTEXT" get tenants.minio.min.io "$name" -n "$namespace" -o json 2>/dev/null || true)"
        if [ -n "$json" ] && printf '%s\n' "$json" | jq -e . >/dev/null 2>&1; then
            state="$(printf '%s\n' "$json" | jq -r '.status.currentState // ""')"
            if [ "$state" = "Initialized" ]; then
                echo "MinIO Tenant $namespace/$name initialized"
                return
            fi
        fi

        attempt=$((attempt + 1))
        if [ "$attempt" -ge 180 ]; then
            kubectl --context "$KUBE_CONTEXT" describe tenants.minio.min.io "$name" -n "$namespace" || true
            exit 1
        fi
        sleep 5
    done
}

cd "$ROOT_DIR"

kubectl --context "$KUBE_CONTEXT" apply -k platform/postgresql/resources
kubectl --context "$KUBE_CONTEXT" apply -k platform/redis/resources
kubectl --context "$KUBE_CONTEXT" apply -k platform/kafka/resources
kubectl --context "$KUBE_CONTEXT" apply -k platform/clickhouse/resources
kubectl --context "$KUBE_CONTEXT" apply -k platform/minio/resources

kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready certificate/redis-main-tls -n monadiam-data --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=create secret/redis-main-auth -n monadiam-data --timeout=180s
helm_upgrade_install redis-main "$REDIS_CHART" \
    --version "$REDIS_CHART_VERSION" \
    --namespace monadiam-data \
    --create-namespace \
    --kube-context "$KUBE_CONTEXT" \
    --values platform/redis/helm/values.yaml \
    --wait \
    --timeout 600s \
    --hide-notes

kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready certificate/minio-main-tls -n monadiam-data --timeout=180s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready cluster.postgresql.cnpg.io/postgresql-main -n monadiam-data --timeout=600s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready kafka.kafka.strimzi.io/kafka-main -n monadiam-messaging --timeout=900s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready kafkatopic.kafka.strimzi.io/connect-configs -n monadiam-messaging --timeout=300s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready kafkatopic.kafka.strimzi.io/connect-offsets -n monadiam-messaging --timeout=300s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready kafkatopic.kafka.strimzi.io/connect-status -n monadiam-messaging --timeout=300s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready kafkauser.kafka.strimzi.io/kafka-main-admin -n monadiam-messaging --timeout=300s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready kafkauser.kafka.strimzi.io/kafka-connect -n monadiam-messaging --timeout=300s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready kafkaconnect.kafka.strimzi.io/kafka-connect -n monadiam-messaging --timeout=900s
kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready kafkaconnector.kafka.strimzi.io/postgresql-main-outbox -n monadiam-messaging --timeout=300s
wait_clickhouse_installation_completed monadiam-data clickhouse-main
wait_minio_tenant_initialized monadiam-data minio-main
bootstrap_minio

kubectl --context "$KUBE_CONTEXT" get cluster.postgresql.cnpg.io -n monadiam-data
kubectl --context "$KUBE_CONTEXT" get pods,svc -l app.kubernetes.io/instance=redis-main -n monadiam-data
kubectl --context "$KUBE_CONTEXT" get kafka.kafka.strimzi.io,kafkanodepool.kafka.strimzi.io,kafkatopic.kafka.strimzi.io,kafkauser.kafka.strimzi.io,kafkaconnect.kafka.strimzi.io,kafkaconnector.kafka.strimzi.io -n monadiam-messaging
kubectl --context "$KUBE_CONTEXT" get clickhouseinstallations.clickhouse.altinity.com -n monadiam-data
kubectl --context "$KUBE_CONTEXT" get tenants.minio.min.io -n monadiam-data
