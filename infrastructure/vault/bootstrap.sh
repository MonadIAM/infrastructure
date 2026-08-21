#!/bin/sh
set -eu

: "${VAULT_ADDR:?}"
: "${VAULT_CACERT:?}"
: "${VAULT_BOOTSTRAP_TOKEN_FILE:?}"
: "${VAULT_AGENT_ROLE_NAME:?}"
: "${VAULT_AGENT_ROLE_ID_FILE:?}"
: "${VAULT_AGENT_SECRET_ID_FILE:?}"

export VAULT_ADDR
export VAULT_CACERT

wait_for_file() {
    path="$1"
    i=0
    until [ -s "$path" ]; do
        i=$((i + 1))
        if [ "$i" -ge 60 ]; then
            echo "Timed out waiting for $path" >&2
            exit 1
        fi
        sleep 1
    done
}

wait_for_file "$VAULT_BOOTSTRAP_TOKEN_FILE"

VAULT_TOKEN="$(cat "$VAULT_BOOTSTRAP_TOKEN_FILE")"
export VAULT_TOKEN

until vault status >/dev/null 2>&1; do
    sleep 1
done

role_path="auth/approle/role/$VAULT_AGENT_ROLE_NAME"
role_id_dir="$(dirname "$VAULT_AGENT_ROLE_ID_FILE")"
secret_id_dir="$(dirname "$VAULT_AGENT_SECRET_ID_FILE")"

mkdir -p "$role_id_dir" "$secret_id_dir"

vault read -field=role_id "$role_path/role-id" > "$VAULT_AGENT_ROLE_ID_FILE"
vault write -f -field=secret_id "$role_path/secret-id" > "$VAULT_AGENT_SECRET_ID_FILE"
chmod 0400 "$VAULT_AGENT_ROLE_ID_FILE" "$VAULT_AGENT_SECRET_ID_FILE"
