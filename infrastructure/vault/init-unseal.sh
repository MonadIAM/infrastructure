#!/bin/sh
set -eu

: "${VAULT_ADDR:?}"
: "${VAULT_CACERT:?}"
: "${VAULT_INIT_KEYS_DIR:?}"

export VAULT_ADDR
export VAULT_CACERT

ROOT_TOKEN_FILE="$VAULT_INIT_KEYS_DIR/root-token"
UNSEAL_KEY_FILE="$VAULT_INIT_KEYS_DIR/unseal-key-1"

mkdir -p "$VAULT_INIT_KEYS_DIR"
chmod 0700 "$VAULT_INIT_KEYS_DIR"

json_bool() {
    key="$1"
    sed -n "s/.*\"$key\"[[:space:]]*:[[:space:]]*\(true\|false\).*/\1/p"
}

json_root_token() {
    sed -n 's/.*"root_token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p'
}

json_first_unseal_key() {
    awk '
        /"unseal_keys_b64"[[:space:]]*:/ { in_keys = 1; next }
        in_keys == 1 && /"/ {
            value = $0
            sub(/^[[:space:]]*"/, "", value)
            sub(/"[,[:space:]]*$/, "", value)
            print value
            exit
        }
    '
}

while :; do
    status_json="$(vault status -format=json 2>/dev/null || true)"

    if [ -z "$status_json" ]; then
        echo "$(date) Vault API is not reachable yet."
        sleep 5
        continue
    fi

    initialized="$(printf '%s\n' "$status_json" | json_bool initialized)"
    sealed="$(printf '%s\n' "$status_json" | json_bool sealed)"

    if [ -z "$initialized" ] || [ -z "$sealed" ]; then
        echo "$(date) Vault status response is not parseable." >&2
        sleep 5
        continue
    fi

    if [ "$initialized" = "false" ]; then
        if [ -e "$ROOT_TOKEN_FILE" ] || [ -e "$UNSEAL_KEY_FILE" ]; then
            echo "$(date) Vault is uninitialized, but init key files already exist in $VAULT_INIT_KEYS_DIR; refusing to re-init against possibly different storage." >&2
            sleep 30
            continue
        fi

        echo "$(date) Initializing Vault with one local Shamir share."
        init_json="$(vault operator init -format=json -key-shares=1 -key-threshold=1)"
        unseal_key="$(printf '%s\n' "$init_json" | json_first_unseal_key)"
        root_token="$(printf '%s\n' "$init_json" | json_root_token)"

        if [ -z "$unseal_key" ] || [ -z "$root_token" ]; then
            echo "$(date) Vault init output did not contain expected key material." >&2
            exit 1
        fi

        umask 077
        printf '%s\n' "$unseal_key" > "$UNSEAL_KEY_FILE"
        printf '%s\n' "$root_token" > "$ROOT_TOKEN_FILE"
        chmod 0400 "$UNSEAL_KEY_FILE" "$ROOT_TOKEN_FILE"
        echo "$(date) Vault initialized; local init keys written to $VAULT_INIT_KEYS_DIR."
        sealed="true"
    fi

    if [ "$sealed" = "true" ]; then
        if [ ! -s "$UNSEAL_KEY_FILE" ]; then
            echo "$(date) Vault is sealed and $UNSEAL_KEY_FILE is missing." >&2
            sleep 10
            continue
        fi

        echo "$(date) Unsealing Vault."
        vault operator unseal "$(cat "$UNSEAL_KEY_FILE")" >/dev/null
    fi

    sleep 10
done
