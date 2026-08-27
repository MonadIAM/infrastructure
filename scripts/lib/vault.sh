require_vault_env() {
    require_env \
        KUBE_CONTEXT \
        HASHICORP_HELM_REPO \
        VAULT_CHART_VERSION \
        VAULT_SECRETS_OPERATOR_VERSION \
        VAULT_NAMESPACE \
        VAULT_POD \
        VAULT_ADDR \
        VAULT_CACERT \
        VAULT_STATE_DIR \
        VAULT_INIT_FILE \
        VAULT_KEY_SHARES \
        VAULT_KEY_THRESHOLD \
        VAULT_POLICY_NAME \
        VAULT_KV_MOUNT \
        VAULT_K8S_AUTH_MOUNT \
        VAULT_K8S_ROLE \
        VAULT_K8S_AUDIENCE \
        VAULT_K8S_TTL \
        VAULT_DATA_POLICY_NAME \
        VAULT_DATA_K8S_ROLE \
        VAULT_MESSAGING_POLICY_NAME \
        VAULT_MESSAGING_K8S_ROLE \
        VAULT_OBSERVABILITY_POLICY_NAME \
        VAULT_OBSERVABILITY_K8S_ROLE \
        VAULT_CLICKHOUSE_OPERATOR_POLICY_NAME \
        VAULT_CLICKHOUSE_OPERATOR_K8S_ROLE \
        GRAFANA_ADMIN_USER \
        MINIO_ARCHIVER_ACCESS_KEY
}

vault_exec() {
    kubectl --context "$KUBE_CONTEXT" exec -n "$VAULT_NAMESPACE" "$VAULT_POD" -- \
        env VAULT_ADDR="$VAULT_ADDR" VAULT_CACERT="$VAULT_CACERT" "$@"
}

vault_exec_with_token() {
    token="$(jq -er '.root_token' "$VAULT_INIT_FILE")"
    printf '%s\n' "$token" | kubectl --context "$KUBE_CONTEXT" exec -i -n "$VAULT_NAMESPACE" "$VAULT_POD" -- \
        sh -c '
        IFS= read -r VAULT_TOKEN
        VAULT_ADDR="$1"
        VAULT_CACERT="$2"
        export VAULT_TOKEN VAULT_ADDR VAULT_CACERT
        shift 2
        exec "$@"
        ' sh "$VAULT_ADDR" "$VAULT_CACERT" "$@"
}

vault_exec_with_token_and_file() {
    input_file="$1"
    shift
    token="$(jq -er '.root_token' "$VAULT_INIT_FILE")"
    {
        printf '%s\n' "$token"
        sed -n '1,$p' "$input_file"
    } | kubectl --context "$KUBE_CONTEXT" exec -i -n "$VAULT_NAMESPACE" "$VAULT_POD" -- \
        sh -c '
        IFS= read -r VAULT_TOKEN
        VAULT_ADDR="$1"
        VAULT_CACERT="$2"
        export VAULT_TOKEN VAULT_ADDR VAULT_CACERT
        shift 2
        exec "$@"
        ' sh "$VAULT_ADDR" "$VAULT_CACERT" "$@"
}

vault_initialized() {
    output="$(vault_exec vault operator init -status 2>&1)" && return 0
    case "$output" in
        *"Vault is not initialized"*)
            return 1
            ;;
        *)
            printf '%s\n' "$output" >&2
            exit 1
            ;;
    esac
}

vault_status_json() {
    output="$(vault_exec sh -c '
    vault status -format=json
    status="$?"
    if [ "$status" -eq 0 ] || [ "$status" -eq 2 ]; then
        exit 0
    fi
    exit "$status"
    ' 2>&1)" || {
        printf '%s\n' "$output" >&2
        return 1
    }

    if printf '%s\n' "$output" | jq -e . >/dev/null 2>&1; then
        printf '%s\n' "$output"
        return 0
    fi

    printf '%s\n' "$output" >&2
    return 1
}

vault_kv_path_exists() {
    vault_exec_with_token vault kv get "${VAULT_KV_MOUNT}/$1" >/dev/null 2>&1
}

vault_random_base64() {
    vault_exec_with_token vault write -field=random_bytes sys/tools/random/32 format=base64
}

vault_random_hex() {
    vault_exec_with_token vault write -field=random_bytes sys/tools/random/32 format=hex
}

vault_kv_field() {
    path="$1"
    field="$2"

    vault_exec_with_token vault kv get -field="$field" "${VAULT_KV_MOUNT}/$path"
}

is_url_userinfo_safe() {
    value="$1"
    unsafe="$(printf '%s' "$value" | tr -d 'A-Za-z0-9._~-')"
    [ -z "$unsafe" ]
}

sha256_hex() {
    value="$1"

    if command -v openssl >/dev/null 2>&1; then
        printf '%s' "$value" | openssl dgst -sha256 -r | awk '{print $1}'
        return
    fi

    if command -v shasum >/dev/null 2>&1; then
        printf '%s' "$value" | shasum -a 256 | awk '{print $1}'
        return
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        printf '%s' "$value" | sha256sum | awk '{print $1}'
        return
    fi

    echo "sha256 helper not found" >&2
    exit 1
}

ensure_basic_auth_credentials() {
    path="$1"
    username="$2"

    if vault_kv_path_exists "$path"; then
        return
    fi

    password="$(vault_random_hex)"
    vault_exec_with_token vault kv put "${VAULT_KV_MOUNT}/$path" \
        username="$username" \
        password="$password"
}

ensure_user_password_credentials() {
    path="$1"
    username="$2"

    if vault_kv_path_exists "$path"; then
        return
    fi

    password="$(vault_random_hex)"
    vault_exec_with_token vault kv put "${VAULT_KV_MOUNT}/$path" \
        username="$username" \
        password="$password"
}

ensure_clickhouse_operator_credentials() {
    path=clickhouse-operator/operator-credentials

    if vault_kv_path_exists "$path"; then
        password="$(vault_kv_field "$path" password)"
        if is_url_userinfo_safe "$password"; then
            return
        fi
    fi

    password="$(vault_random_hex)"
    vault_exec_with_token vault kv put "${VAULT_KV_MOUNT}/$path" \
        username=clickhouse_operator \
        password="$password"
}

ensure_clickhouse_app_credentials() {
    if vault_kv_path_exists monadiam-data/clickhouse/main/app; then
        return
    fi

    password="$(vault_random_hex)"
    password_sha256_hex="$(sha256_hex "$password")"
    vault_exec_with_token vault kv put "${VAULT_KV_MOUNT}/monadiam-data/clickhouse/main/app" \
        username=app \
        password_sha256_hex="$password_sha256_hex"
}

ensure_clickhouse_intercluster_secret() {
    if vault_kv_path_exists monadiam-data/clickhouse/main/intercluster; then
        return
    fi

    secret="$(vault_random_hex)"
    vault_exec_with_token vault kv put "${VAULT_KV_MOUNT}/monadiam-data/clickhouse/main/intercluster" \
        secret="$secret"
}

ensure_minio_root_credentials() {
    if vault_kv_path_exists monadiam-data/minio/main/root; then
        return
    fi

    password="$(vault_random_hex)"
    vault_exec_with_token vault kv put "${VAULT_KV_MOUNT}/monadiam-data/minio/main/root" \
        MINIO_ROOT_USER=minioadmin \
        MINIO_ROOT_PASSWORD="$password"
}

ensure_redis_main_credentials() {
    if vault_kv_path_exists monadiam-data/redis/main/auth; then
        return
    fi

    password="$(vault_random_hex)"
    vault_exec_with_token vault kv put "${VAULT_KV_MOUNT}/monadiam-data/redis/main/auth" \
        redis-password="$password"
}

ensure_grafana_admin_credentials() {
    if vault_kv_path_exists monadiam-observability/grafana/admin; then
        return
    fi

    password="$(vault_random_hex)"
    vault_exec_with_token vault kv put "${VAULT_KV_MOUNT}/monadiam-observability/grafana/admin" \
        admin-user="$GRAFANA_ADMIN_USER" \
        admin-password="$password"
}

ensure_minio_archiver_credentials() {
    if vault_kv_path_exists monadiam-data/minio/main/archiver; then
        return
    fi

    secret_key="$(vault_random_hex)"
    vault_exec_with_token vault kv put "${VAULT_KV_MOUNT}/monadiam-data/minio/main/archiver" \
        MINIO_ARCHIVER_ACCESS_KEY="$MINIO_ARCHIVER_ACCESS_KEY" \
        MINIO_ARCHIVER_SECRET_KEY="$secret_key"
}

ensure_stateful_cluster_credentials() {
    ensure_basic_auth_credentials monadiam-data/postgresql/main/superuser postgres
    ensure_basic_auth_credentials monadiam-data/postgresql/main/app app
    ensure_basic_auth_credentials monadiam-data/postgresql/main/debezium debezium
    ensure_user_password_credentials monadiam-messaging/kafka/main/admin kafka-main-admin
    ensure_clickhouse_app_credentials
    ensure_clickhouse_intercluster_secret
    ensure_redis_main_credentials
    ensure_minio_root_credentials
    ensure_minio_archiver_credentials
}

initialize_vault() {
    if vault_initialized; then
        echo "Vault is already initialized."
        return
    fi

    umask 077
    mkdir -p "$VAULT_STATE_DIR"
    chmod 700 "$VAULT_STATE_DIR"

    if [ -f "$VAULT_INIT_FILE" ]; then
        backup_file="$VAULT_INIT_FILE.$(date +%Y%m%d%H%M%S).bak"
        mv "$VAULT_INIT_FILE" "$backup_file"
        chmod 600 "$backup_file"
        echo "Existing Vault init file moved to $backup_file"
    fi

    tmp_file="$VAULT_INIT_FILE.tmp"
    vault_exec vault operator init \
        -key-shares="$VAULT_KEY_SHARES" \
        -key-threshold="$VAULT_KEY_THRESHOLD" \
        -format=json > "$tmp_file"
    chmod 600 "$tmp_file"
    mv "$tmp_file" "$VAULT_INIT_FILE"
    echo "Vault init material stored at $VAULT_INIT_FILE"
}

unseal_vault() {
    status="$(vault_status_json)"
    sealed="$(printf '%s\n' "$status" | jq -r '.sealed')"

    if [ "$sealed" = "false" ]; then
        echo "Vault is already unsealed."
        return
    fi

    if [ ! -f "$VAULT_INIT_FILE" ]; then
        echo "Vault is sealed and local init file is missing: $VAULT_INIT_FILE" >&2
        echo "Reset the local cluster or restore the matching init file outside the repository." >&2
        exit 1
    fi

    unseal_key="$(jq -er '.unseal_keys_b64[0]' "$VAULT_INIT_FILE")"
    printf '%s\n' "$unseal_key" | kubectl --context "$KUBE_CONTEXT" exec -i -n "$VAULT_NAMESPACE" "$VAULT_POD" -- \
        sh -c '
        IFS= read -r unseal_key
        VAULT_ADDR="$1"
        VAULT_CACERT="$2"
        export VAULT_ADDR VAULT_CACERT
        vault operator unseal "$unseal_key"
        ' sh "$VAULT_ADDR" "$VAULT_CACERT"
}

bootstrap_vault() {
    if [ ! -f "$VAULT_INIT_FILE" ]; then
        echo "Vault init file not found at $VAULT_INIT_FILE; cannot run local bootstrap." >&2
        echo "Restore the matching init file outside the repository or reset the local cluster." >&2
        exit 1
    fi

    if ! vault_exec_with_token vault secrets list -format=json | grep -q "\"${VAULT_KV_MOUNT}/\""; then
        vault_exec_with_token vault secrets enable -path="$VAULT_KV_MOUNT" kv-v2
    fi

    if ! vault_exec_with_token vault auth list -format=json | grep -q "\"${VAULT_K8S_AUTH_MOUNT}/\""; then
        vault_exec_with_token vault auth enable -path="$VAULT_K8S_AUTH_MOUNT" kubernetes
    fi

    vault_exec_with_token vault write "auth/${VAULT_K8S_AUTH_MOUNT}/config" \
        kubernetes_host='https://kubernetes.default.svc:443'

    vault_exec_with_token_and_file "$ROOT_DIR/platform/vault/bootstrap/monadiam-apps-read.hcl" \
        vault policy write "$VAULT_POLICY_NAME" -
    vault_exec_with_token_and_file "$ROOT_DIR/platform/vault/bootstrap/monadiam-data-read.hcl" \
        vault policy write "$VAULT_DATA_POLICY_NAME" -
    vault_exec_with_token_and_file "$ROOT_DIR/platform/vault/bootstrap/monadiam-messaging-read.hcl" \
        vault policy write "$VAULT_MESSAGING_POLICY_NAME" -
    vault_exec_with_token_and_file "$ROOT_DIR/platform/vault/bootstrap/monadiam-observability-read.hcl" \
        vault policy write "$VAULT_OBSERVABILITY_POLICY_NAME" -
    vault_exec_with_token_and_file "$ROOT_DIR/platform/vault/bootstrap/clickhouse-operator-read.hcl" \
        vault policy write "$VAULT_CLICKHOUSE_OPERATOR_POLICY_NAME" -

    vault_exec_with_token vault write "auth/${VAULT_K8S_AUTH_MOUNT}/role/${VAULT_K8S_ROLE}" \
        bound_service_account_names=vault-auth \
        bound_service_account_namespaces=monadiam-apps \
        policies="$VAULT_POLICY_NAME" \
        audience="$VAULT_K8S_AUDIENCE" \
        ttl="$VAULT_K8S_TTL"
    vault_exec_with_token vault write "auth/${VAULT_K8S_AUTH_MOUNT}/role/${VAULT_DATA_K8S_ROLE}" \
        bound_service_account_names=vault-auth \
        bound_service_account_namespaces=monadiam-data \
        policies="$VAULT_DATA_POLICY_NAME" \
        audience="$VAULT_K8S_AUDIENCE" \
        ttl="$VAULT_K8S_TTL"
    vault_exec_with_token vault write "auth/${VAULT_K8S_AUTH_MOUNT}/role/${VAULT_MESSAGING_K8S_ROLE}" \
        bound_service_account_names=vault-auth \
        bound_service_account_namespaces=monadiam-messaging \
        policies="$VAULT_MESSAGING_POLICY_NAME" \
        audience="$VAULT_K8S_AUDIENCE" \
        ttl="$VAULT_K8S_TTL"
    vault_exec_with_token vault write "auth/${VAULT_K8S_AUTH_MOUNT}/role/${VAULT_OBSERVABILITY_K8S_ROLE}" \
        bound_service_account_names=vault-auth \
        bound_service_account_namespaces=monadiam-observability \
        policies="$VAULT_OBSERVABILITY_POLICY_NAME" \
        audience="$VAULT_K8S_AUDIENCE" \
        ttl="$VAULT_K8S_TTL"
    vault_exec_with_token vault write "auth/${VAULT_K8S_AUTH_MOUNT}/role/${VAULT_CLICKHOUSE_OPERATOR_K8S_ROLE}" \
        bound_service_account_names=vault-auth \
        bound_service_account_namespaces=clickhouse-operator \
        policies="$VAULT_CLICKHOUSE_OPERATOR_POLICY_NAME" \
        audience="$VAULT_K8S_AUDIENCE" \
        ttl="$VAULT_K8S_TTL"

    if ! vault_exec_with_token vault audit list -format=json | grep -q '"file/"'; then
        vault_exec_with_token vault audit enable file file_path=/vault/audit/audit.log
    fi

    ensure_clickhouse_operator_credentials
    ensure_stateful_cluster_credentials
    ensure_grafana_admin_credentials

    vault_exec_with_token vault kv put "${VAULT_KV_MOUNT}/monadiam-apps/bootstrap" status=configured
    vault_exec_with_token vault kv put "${VAULT_KV_MOUNT}/monadiam-data/bootstrap" status=configured
    vault_exec_with_token vault kv put "${VAULT_KV_MOUNT}/monadiam-messaging/bootstrap" status=configured
    vault_exec_with_token vault kv put "${VAULT_KV_MOUNT}/monadiam-observability/bootstrap" status=configured
}

wait_vault_static_secret_reconciled() {
    namespace="$1"
    name="$2"
    destination_secret="$3"
    attempt=0

    while :; do
        json="$(kubectl --context "$KUBE_CONTEXT" get vaultstaticsecrets.secrets.hashicorp.com "$name" -n "$namespace" -o json 2>/dev/null || true)"
        if [ -n "$json" ] && printf '%s\n' "$json" | jq -e . >/dev/null 2>&1; then
            generation="$(printf '%s\n' "$json" | jq -r '.metadata.generation')"
            last_generation="$(printf '%s\n' "$json" | jq -r '.status.lastGeneration // 0')"
            ready="$(printf '%s\n' "$json" | jq -r '[.status.conditions[]? | select(.type == "Ready")][-1].status // ""')"
            healthy="$(printf '%s\n' "$json" | jq -r '[.status.conditions[]? | select(.type == "Healthy")][-1].status // ""')"
            secret_mac="$(printf '%s\n' "$json" | jq -r '.status.secretMAC // ""')"

            if [ "$generation" = "$last_generation" ] &&
                    [ "$ready" = "True" ] &&
                    [ "$healthy" = "True" ] &&
                    [ -n "$secret_mac" ] &&
                    kubectl --context "$KUBE_CONTEXT" get secret "$destination_secret" -n "$namespace" >/dev/null 2>&1; then
                echo "VaultStaticSecret $namespace/$name reconciled"
                return
            fi
        fi

        attempt=$((attempt + 1))
        if [ "$attempt" -ge 36 ]; then
            kubectl --context "$KUBE_CONTEXT" describe vaultstaticsecrets.secrets.hashicorp.com "$name" -n "$namespace" || true
            kubectl --context "$KUBE_CONTEXT" get secret "$destination_secret" -n "$namespace" || true
            exit 1
        fi
        sleep 5
    done
}
