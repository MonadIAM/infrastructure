#!/bin/sh

wait_for_vault() {
    i=0
    until vault status >/dev/null 2>&1; do
        i=$((i + 1))
        if [ "$i" -ge 60 ]; then
            echo "Vault API is not reachable at $VAULT_ADDR" >&2
            exit 1
        fi
        sleep 1
    done
}

enable_secret_engine() {
    path="$1"
    type="$2"

    vault secrets list -format=json | grep -q "\"$path/\"" \
        || vault secrets enable -path="$path" "$type" >/dev/null
}

enable_auth_method() {
    path="$1"
    type="$2"

    vault auth list -format=json | grep -q "\"$path/\"" \
        || vault auth enable -path="$path" "$type" >/dev/null
}

write_policy() {
    name="$1"

    vault policy write "$name" "/vault/policies/$name.hcl" >/dev/null
}

create_key() {
    mount="$1"
    name="$2"
    type="$3"
    shift 3

    vault read "$mount/keys/$name" >/dev/null 2>&1 \
        || vault write -f "$mount/keys/$name" type="$type" "$@" >/dev/null
}

create_signing_key() {
    mount="$1"
    name="$2"
    type="$3"

    create_key "$mount" "$name" "$type"
    vault write "$mount/keys/$name/config" deletion_allowed=true >/dev/null
}

configure_approle() {
    role="$1"
    policies="$2"

    vault write "auth/approle/role/$role" \
        token_policies="$policies" \
        token_ttl=1h \
        token_max_ttl=24h \
        secret_id_ttl=0 \
        secret_id_num_uses=0 >/dev/null
}
