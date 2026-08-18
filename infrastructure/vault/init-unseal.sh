#!/bin/sh
set -eu

KEYS_DIR=/vault/init-keys
ROOT_TOKEN_FILE="$KEYS_DIR/root-token"
UNSEAL_KEY_FILE="$KEYS_DIR/unseal-key-1"

mkdir -p "$KEYS_DIR"

while :; do
    status_json="$(vault status -format=json 2>/dev/null || true)"

    if [ -z "$status_json" ]; then
        echo "$(date) Vault API not reachable yet, retrying..."
        sleep 5
        continue
    fi

    initialized="$(printf '%s\n' "$status_json" | sed -n 's/.*"initialized"[[:space:]]*:[[:space:]]*\(true\|false\).*/\1/p')"
    sealed="$(printf '%s\n' "$status_json" | sed -n 's/.*"sealed"[[:space:]]*:[[:space:]]*\(true\|false\).*/\1/p')"

    if [ -z "$initialized" ] || [ -z "$sealed" ]; then
        echo "$(date) Could not parse Vault status response, retrying..." >&2
        sleep 5
        continue
    fi

    if [ "$initialized" != "true" ]; then
        if [ -f "$ROOT_TOKEN_FILE" ] && [ -f "$UNSEAL_KEY_FILE" ]; then
            echo "$(date) Vault reports uninitialized but $KEYS_DIR already has key files from a previous init - refusing to auto re-init (this would orphan whatever is/was in Raft storage). Remove $KEYS_DIR by hand if this is meant to be a genuine fresh start, then restart this container." >&2
            sleep 30
            continue
        fi

        echo "$(date) Initializing Vault (single Shamir share, threshold 1)..."
        init_output="$(vault operator init -key-shares=1 -key-threshold=1)"
        printf '%s\n' "$init_output" | grep '^Unseal Key 1:' | sed 's/^Unseal Key 1: *//' > "$UNSEAL_KEY_FILE"
        printf '%s\n' "$init_output" | grep '^Initial Root Token:' | sed 's/^Initial Root Token: *//' > "$ROOT_TOKEN_FILE"
        chmod 0400 "$UNSEAL_KEY_FILE" "$ROOT_TOKEN_FILE"
        echo "$(date) Vault initialized. Unseal key and root token written to $KEYS_DIR (local, gitignored - not synced anywhere)."
        sealed="true"
    fi

    if [ "$sealed" = "true" ]; then
        if [ ! -f "$UNSEAL_KEY_FILE" ]; then
            echo "$(date) Vault is sealed but no local unseal key file at $UNSEAL_KEY_FILE - cannot auto-unseal. Manual intervention required (vault operator unseal)." >&2
        else
            echo "$(date) Vault is sealed, unsealing..."
            vault operator unseal "$(cat "$UNSEAL_KEY_FILE")" >/dev/null
            echo "$(date) Vault unsealed."
        fi
    fi

    sleep 10
done
