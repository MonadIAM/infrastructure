cluster_exists() {
    k3d cluster list "$CLUSTER" --no-headers 2>/dev/null |
        awk -v cluster="$CLUSTER" '$1 == cluster { found = 1 } END { exit found ? 0 : 1 }'
}

start_or_create_cluster() {
    require_env CLUSTER K3D_CONFIG KUBE_CONTEXT

    cd "$ROOT_DIR"

    if cluster_exists; then
        echo "k3d cluster '$CLUSTER' already exists; starting it."
        k3d cluster start "$CLUSTER" --timeout 180s
    else
        k3d cluster create --config "$K3D_CONFIG"
    fi

    kubectl config use-context "$KUBE_CONTEXT" >/dev/null
    kubectl --context "$KUBE_CONTEXT" wait --for=condition=Ready node --all --timeout=180s
    kubectl --context "$KUBE_CONTEXT" apply -k "$ROOT_DIR/clusters/local"
    kubectl --context "$KUBE_CONTEXT" get nodes -o wide
}

stop_cluster() {
    require_env CLUSTER

    if cluster_exists; then
        k3d cluster stop "$CLUSTER"
    else
        echo "k3d cluster '$CLUSTER' does not exist"
    fi
}

delete_cluster() {
    require_env CLUSTER

    if cluster_exists; then
        k3d cluster delete "$CLUSTER"
    else
        echo "k3d cluster '$CLUSTER' does not exist."
    fi
}
