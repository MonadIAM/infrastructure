#!/bin/sh
set -eu

VAULT_TOKEN="$(cat /vault/init-keys/root-token)"
export VAULT_TOKEN

i=0
until vault status >/dev/null 2>&1; do
    i=$((i + 1))
    if [ "$i" -ge 60 ]; then
        echo "Vault did not become ready in time ($VAULT_ADDR)" >&2
        exit 1
    fi
    sleep 1
done

i=0
while :; do
    if error_output="$(vault read -field=role_id auth/approle/role/infra/role-id 2>&1 > /vault/config/role-id)"; then
        break
    fi

    case "$error_output" in
        *"Code: 403"*|*"Code: 401"*)
            echo "Vault rejected VAULT_TOKEN while reading AppRole 'infra': $error_output" >&2
            exit 1
            ;;
    esac

    i=$((i + 1))
    if [ "$i" -ge 120 ]; then
        echo "AppRole 'infra' was never provisioned" >&2
        exit 1
    fi
    sleep 2
done

vault write -f -field=secret_id auth/approle/role/infra/secret-id > /vault/agent-bootstrap/secret-id
chmod 0400 /vault/config/role-id /vault/agent-bootstrap/secret-id
