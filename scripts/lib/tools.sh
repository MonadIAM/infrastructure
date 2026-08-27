check_required_tools() {
    missing=0

    require_tool docker
    require_tool k3d
    require_tool kubectl
    require_tool helm
    require_tool jq

    if [ "$missing" -ne 0 ]; then
        exit 1
    fi

    docker info >/dev/null
    k3d version
    kubectl version --client
    helm version --short
    jq --version
}

require_tool() {
    tool="$1"

    if ! command -v "$tool" >/dev/null 2>&1; then
        echo "Missing required tool: $tool" >&2
        missing=1
    fi
}
