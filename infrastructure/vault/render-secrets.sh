#!/bin/sh
set -eu

export VAULT_TOKEN="$(cat /vault/token/secrets)"

write_secret() {
    path="$1"
    field="$2"
    target="$3"

    vault kv get -field="$field" "$VAULT_KV_MOUNT/$path" > "$target"
    chmod 0444 "$target"
}

write_secret "$VAULT_KV_CLICKHOUSE_PATH" clickhouse_password /secrets/clickhouse_password
write_secret "$VAULT_KV_CLICKHOUSE_PATH" clickhouse_writer_password /secrets/clickhouse_writer_password
write_secret "$VAULT_KV_CLICKHOUSE_PATH" clickhouse_reader_password /secrets/clickhouse_reader_password

write_secret "$VAULT_KV_MINIO_PATH" minio_root_password /secrets-minio/minio_root_password
write_secret "$VAULT_KV_MINIO_PATH" minio_archiver_secret_key /secrets-minio/minio_archiver_secret_key

rm -f /vault/token/secrets
