#!/bin/sh
set -eu

echo "Waiting for Vault at $VAULT_ADDR..."
i=0
until vault status >/dev/null 2>&1; do
    i=$((i + 1))
    if [ "$i" -ge 60 ]; then
        echo "Vault did not become ready in time ($VAULT_ADDR)" >&2
        exit 1
    fi
    sleep 1
done

vault secrets list -format=json | grep -q "\"$VAULT_KV_MOUNT/\"" ||
    vault secrets enable -path="$VAULT_KV_MOUNT" -version=2 kv

vault kv get "$VAULT_KV_MOUNT/$VAULT_KV_CLICKHOUSE_PATH" >/dev/null 2>&1 ||
    vault kv put "$VAULT_KV_MOUNT/$VAULT_KV_CLICKHOUSE_PATH" \
        clickhouse_password="$CLICKHOUSE_PASSWORD" \
        clickhouse_writer_password="$CLICKHOUSE_WRITER_PASSWORD" \
        clickhouse_reader_password="$CLICKHOUSE_READER_PASSWORD"

vault kv get "$VAULT_KV_MOUNT/$VAULT_KV_MINIO_PATH" >/dev/null 2>&1 ||
    vault kv put "$VAULT_KV_MOUNT/$VAULT_KV_MINIO_PATH" \
        minio_root_password="$MINIO_ROOT_PASSWORD" \
        minio_archiver_secret_key="$MINIO_ARCHIVER_SECRET_KEY"

sed \
    -e "s|{{kv_mount}}|$VAULT_KV_MOUNT|g" \
    -e "s|{{clickhouse_path}}|$VAULT_KV_CLICKHOUSE_PATH|g" \
    -e "s|{{minio_path}}|$VAULT_KV_MINIO_PATH|g" \
    /secrets-policy.hcl > /tmp/secrets-policy.hcl

vault policy write infra-secrets /tmp/secrets-policy.hcl
vault token create \
    -field=token \
    -policy=infra-secrets \
    -no-default-policy \
    -ttl=15m \
    -explicit-max-ttl=15m \
    > /vault/token/secrets
chmod 0400 /vault/token/secrets
