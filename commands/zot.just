set quiet
set working-directory := '..'
set dotenv-filename := '.env.dev'

# Apply the Zot manifests and wait for the registry Deployment.
apply:
    kubectl --kubeconfig "$KUBECONFIG" --context "$KUBE_CONTEXT" apply -k infrastructure/namespaces/base
    kubectl --kubeconfig "$KUBECONFIG" --context "$KUBE_CONTEXT" apply -k infrastructure/zot/base
    kubectl --kubeconfig "$KUBECONFIG" --context "$KUBE_CONTEXT" \
        -n monadiam-data rollout status deployment/monadiam-registry \
        --timeout=300s

# Tag a local Docker image as the requested Zot path and push it to the registry.
push source target:
    docker tag '{{ source }}' 'localhost:5000/{{ target }}'
    docker push 'localhost:5000/{{ target }}'

# Show the Zot Deployment, pods, and Service.
status:
    kubectl --kubeconfig "$KUBECONFIG" --context "$KUBE_CONTEXT" \
        -n monadiam-data get deployment,pod,service \
        -l app.kubernetes.io/name=monadiam-registry

# Follow the Zot Deployment logs.
logs:
    kubectl --kubeconfig "$KUBECONFIG" --context "$KUBE_CONTEXT" -n monadiam-data logs -f deployment/monadiam-registry
