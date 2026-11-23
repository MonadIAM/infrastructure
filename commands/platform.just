set quiet
set working-directory := '..'

profile := env_var_or_default('PROFILE', 'dev')
env_file := env_var_or_default('ENV_FILE', '.env.' + profile)
kubeconfig := env_var('HOME') + '/.kube/monadiam-microk8s-' + profile + '.yaml'
context := 'monadiam-microk8s-' + profile

# Reconcile one platform component through the Go CLI, or all components by default.
reconcile component='all':
    go run . platform reconcile --component '{{ component }}' --env-file '{{ env_file }}'

# Show a summary of cluster component health.
status:
    go run . cluster status --env-file '{{ env_file }}'

# Restart Istio CNI and wait for its DaemonSet to become ready.
ambient:
    kubectl --kubeconfig '{{ kubeconfig }}' --context '{{ context }}' \
        -n istio-system rollout restart daemonset/istio-cni-node
    kubectl --kubeconfig '{{ kubeconfig }}' --context '{{ context }}' \
        -n istio-system rollout status daemonset/istio-cni-node \
        --timeout=300s

# Wait for Prometheus, Loki, and OTel Argo CD applications to be synced and healthy.
observability-ready:
    kubectl --kubeconfig '{{ kubeconfig }}' --context '{{ context }}' \
        -n argocd wait --for=create \
        application/monadiam-prometheus-stack \
        application/monadiam-loki \
        application/monadiam-otel-collector --timeout=900s
    kubectl --kubeconfig '{{ kubeconfig }}' --context '{{ context }}' \
        -n argocd wait --for=jsonpath='{.status.sync.status}'=Synced \
        application/monadiam-prometheus-stack \
        application/monadiam-loki \
        application/monadiam-otel-collector --timeout=900s
    kubectl --kubeconfig '{{ kubeconfig }}' --context '{{ context }}' \
        -n argocd wait --for=jsonpath='{.status.operationState.phase}'=Succeeded \
        application/monadiam-prometheus-stack \
        application/monadiam-loki \
        application/monadiam-otel-collector --timeout=900s
    kubectl --kubeconfig '{{ kubeconfig }}' --context '{{ context }}' \
        -n argocd wait --for=jsonpath='{.status.health.status}'=Healthy \
        application/monadiam-prometheus-stack \
        application/monadiam-loki \
        application/monadiam-otel-collector --timeout=900s

# Forward Grafana to local port 3000.
grafana-ui:
    kubectl --kubeconfig '{{ kubeconfig }}' --context '{{ context }}' \
        -n monadiam-observability port-forward \
        service/kube-prometheus-stack-grafana 3000:80

# Forward Schema Registry to local port 8081.
schema-ui:
    kubectl --kubeconfig '{{ kubeconfig }}' --context '{{ context }}' \
        -n monadiam-messaging port-forward \
        service/schema-registry-main 8081:8080
