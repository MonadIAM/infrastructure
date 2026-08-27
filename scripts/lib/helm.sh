require_helm_env() {
    require_env \
        GODEBUG \
        HELM_RETRY_ATTEMPTS \
        HELM_RETRY_DELAY_SECONDS \
        HELM_RETRY_MAX_DELAY_SECONDS
}

helm_cmd() {
    GODEBUG="$GODEBUG" helm "$@"
}

helm_retry() {
    require_helm_env

    attempt=1
    max_attempts="$HELM_RETRY_ATTEMPTS"
    delay_seconds="$HELM_RETRY_DELAY_SECONDS"

    while :; do
        if helm_cmd "$@"; then
            return
        fi

        if [ "$attempt" -ge "$max_attempts" ]; then
            return 1
        fi

        echo "Helm command failed; retrying in ${delay_seconds}s (${attempt}/${max_attempts}): helm $*" >&2
        sleep "$delay_seconds"

        attempt=$((attempt + 1))
        delay_seconds=$((delay_seconds * 2))
        if [ "$delay_seconds" -gt "$HELM_RETRY_MAX_DELAY_SECONDS" ]; then
            delay_seconds="$HELM_RETRY_MAX_DELAY_SECONDS"
        fi
    done
}

helm_repo_add() {
    name="$1"
    url="$2"

    helm_retry repo add "$name" "$url" --force-update
}

helm_upgrade_install() {
    helm_retry upgrade --install "$@"
}

helm_show_crds() {
    helm_retry show crds "$@"
}
