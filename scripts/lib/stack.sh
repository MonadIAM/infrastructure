reconcile_platform_stack() {
    "$ROOT_DIR/scripts/platform/cert-manager.sh"
    "$ROOT_DIR/scripts/platform/metrics-server.sh"
    "$ROOT_DIR/scripts/platform/istio.sh"
    "$ROOT_DIR/scripts/platform/gateway.sh"
    "$ROOT_DIR/scripts/platform/vault.sh"
    "$ROOT_DIR/scripts/platform/operators.sh"
    "$ROOT_DIR/scripts/platform/clusters.sh"
    "$ROOT_DIR/scripts/platform/observability.sh"
}

show_cluster_status() {
    require_env KUBE_CONTEXT

    kubectl --context "$KUBE_CONTEXT" get nodes -o wide
    kubectl --context "$KUBE_CONTEXT" get namespaces
    helm --kube-context "$KUBE_CONTEXT" list -A
    kubectl --context "$KUBE_CONTEXT" get cluster.postgresql.cnpg.io -n monadiam-data || true
    kubectl --context "$KUBE_CONTEXT" get pods,svc -l app.kubernetes.io/instance=redis-main -n monadiam-data || true
    kubectl --context "$KUBE_CONTEXT" get kafka.kafka.strimzi.io,kafkanodepool.kafka.strimzi.io,kafkatopic.kafka.strimzi.io,kafkauser.kafka.strimzi.io,kafkaconnect.kafka.strimzi.io,kafkaconnector.kafka.strimzi.io -n monadiam-messaging || true
    kubectl --context "$KUBE_CONTEXT" get clickhouseinstallations.clickhouse.altinity.com -n monadiam-data || true
    kubectl --context "$KUBE_CONTEXT" get tenants.minio.min.io -n monadiam-data || true
    kubectl --context "$KUBE_CONTEXT" get pods,svc -n monadiam-observability || true
}
