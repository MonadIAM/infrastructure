set quiet
set working-directory := '..'

profile := env_var_or_default('PROFILE', 'dev')
kubeconfig := env_var('HOME') + '/.kube/monadiam-microk8s-' + profile + '.yaml'
context := 'monadiam-microk8s-' + profile

# Render Kubernetes manifests and validate their schemas and Helm charts.
validate:
    bash scripts/manifests.check.sh

# Start the prepared MicroK8s node and refresh the operator kubeconfig and system settings.
start:
    sudo microk8s start
    sudo microk8s status --wait-ready
    sudo microk8s kubectl wait --for=condition=Ready node/monadiam-{{ profile }} --timeout=180s
    mkdir -p '{{ env_var("HOME") }}/.kube'
    sudo microk8s config > '{{ kubeconfig }}'
    chmod 600 '{{ kubeconfig }}'
    kubectl --kubeconfig '{{ kubeconfig }}' config rename-context microk8s '{{ context }}'
    node_ip=$(sudo microk8s kubectl get node monadiam-{{ profile }} \
        -o jsonpath='{.status.addresses[?(@.type=="InternalIP")].address}'); \
        kubectl --kubeconfig '{{ kubeconfig }}' config set-cluster \
        microk8s-cluster --server="https://$node_ip:16443"
    kubectl --kubeconfig '{{ kubeconfig }}' apply -f clusters/{{ profile }}/infrastructure/system/limits.yaml
    kubectl --kubeconfig '{{ kubeconfig }}' -n kube-system \
        patch deployment coredns --type=strategic \
        --patch-file clusters/{{ profile }}/infrastructure/system/coredns.yaml
    kubectl --kubeconfig '{{ kubeconfig }}' -n kube-system \
        patch deployment metrics-server --type=strategic \
        --patch-file clusters/{{ profile }}/infrastructure/system/metrics-server.yaml
    kubectl --kubeconfig '{{ kubeconfig }}' -n kube-system rollout status deployment/coredns --timeout=180s

# Stop MicroK8s on the current host.
stop:
    sudo microk8s stop

# Show MicroK8s status and enabled addons.
status:
    microk8s status

# Show cluster nodes, pods, PVCs, and recent events without changing resources.
diagnostics:
    kubectl --kubeconfig '{{ kubeconfig }}' --context '{{ context }}' get nodes
    kubectl --kubeconfig '{{ kubeconfig }}' --context '{{ context }}' get pods -A
    kubectl --kubeconfig '{{ kubeconfig }}' --context '{{ context }}' get pvc -A
    kubectl --kubeconfig '{{ kubeconfig }}' --context '{{ context }}' get events -A --sort-by=.lastTimestamp
