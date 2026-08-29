#!/bin/sh
set -eu

ROOT_TOKEN_FILE="$VAULT_INIT_DIR/root-token"
UNSEAL_KEY_FILE="$VAULT_INIT_DIR/unseal-key"

mkdir -p "$VAULT_INIT_DIR"
chmod 0700 "$VAULT_INIT_DIR"

wait_for_status() {
    i=0
    while true; do
        set +e
        vault status >/tmp/vault-status 2>&1
        code=$?
        set -e
        if [ "$code" -eq 0 ] || [ "$code" -eq 2 ]; then
            break
        fi

        i=$((i + 1))
        if [ "$i" -ge 60 ]; then
            echo "Vault API is not reachable at $VAULT_ADDR" >&2
            cat /tmp/vault-status >&2 || true
            exit 1
        fi
        sleep 1
    done
}

is_initialized() {
    vault status | awk '/Initialized/ { print $2 }'
}

is_sealed() {
    vault status | awk '/Sealed/ { print $2 }'
}

wait_for_status

if [ "$(is_initialized)" = "false" ]; then
    if [ -e "$ROOT_TOKEN_FILE" ] || [ -e "$UNSEAL_KEY_FILE" ]; then
        echo "Vault is uninitialized, but init files already exist in $VAULT_INIT_DIR." >&2
        exit 1
    fi

    vault operator init -key-shares=1 -key-threshold=1 > /tmp/vault-init
    awk -F': ' '/Unseal Key 1/ { print $2 }' /tmp/vault-init > "$UNSEAL_KEY_FILE"
    awk -F': ' '/Initial Root Token/ { print $2 }' /tmp/vault-init > "$ROOT_TOKEN_FILE"
    chmod 0400 "$UNSEAL_KEY_FILE" "$ROOT_TOKEN_FILE"
fi

if [ "$(is_sealed)" = "true" ]; then
    if [ ! -s "$UNSEAL_KEY_FILE" ]; then
        echo "Vault is sealed and $UNSEAL_KEY_FILE is missing." >&2
        exit 1
    fi

    vault operator unseal "$(cat "$UNSEAL_KEY_FILE")" >/dev/null
fi

vault status >/dev/null
ROOT_VAULT_TOKEN="$(cat "$ROOT_TOKEN_FILE")"
LOCAL_VAULT_TOKEN="$VAULT_TOKEN"
export VAULT_TOKEN="$ROOT_VAULT_TOKEN"

if [ "$LOCAL_VAULT_TOKEN" != "$ROOT_VAULT_TOKEN" ]; then
    vault token revoke "$LOCAL_VAULT_TOKEN" >/dev/null 2>&1 || true
    vault token create -id="$LOCAL_VAULT_TOKEN" -policy=root -ttl=0 >/dev/null
fi

echo "Vault is initialized and unsealed."
