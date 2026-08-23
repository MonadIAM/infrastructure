#!/bin/sh
set -eu

: "${SERVICE_CA_BUNDLE_FILE:?}"
: "${KAFKA_SECRET_GROUP_ID:?}"
: "${SERVICE_CERT_DIR:?}"
: "${VAULT_TOKEN_FILE:?}"
: "${SERVICE_CA_DIR:?}"
: "${VAULT_CACERT:?}"
: "${VAULT_ADDR:?}"
: "${PKI_MOUNT:?}"
: "${TLS_TTL:?}"

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

json_string_field() {
    key="$1"
    awk -v key="$key" '
        {
            marker = "\"" key "\":"
            pos = index($0, marker)
            if (pos == 0) {
                next
            }

            value = substr($0, pos + length(marker))
            sub(/^[[:space:]]*\"/, "", value)
            sub(/\",[[:space:]]*$/, "", value)
            sub(/\"$/, "", value)
            gsub(/\\n/, "\n", value)
            gsub(/\\r/, "\r", value)
            gsub(/\\\"/, "\"", value)
            gsub(/\\\\/, "\\", value)
            print value
            found = 1
        }
        END {
            if (found != 1) {
                exit 1
            }
        }
    '
}

issue_cert() {
    name="$1"
    role="$2"
    common_name="$3"
    alt_names="$4"
    ip_sans="$5"

    set -- "$PKI_MOUNT/issue/$role" "common_name=$common_name" "ttl=$TLS_TTL" "private_key_format=pkcs8"
    if [ -n "$alt_names" ]; then
        set -- "$@" "alt_names=$alt_names"
    fi
    if [ -n "$ip_sans" ]; then
        set -- "$@" "ip_sans=$ip_sans"
    fi

    response="$(vault write -format=json "$@")"
    printf '%s\n' "$response" | json_string_field certificate > "$SERVICE_CERT_DIR/$name.crt"
    printf '%s\n' "$response" | json_string_field private_key > "$SERVICE_CERT_DIR/$name.key"
    cat "$SERVICE_CERT_DIR/$name.key" "$SERVICE_CERT_DIR/$name.crt" > "$SERVICE_CERT_DIR/$name.pem"
    chmod 0444 "$SERVICE_CERT_DIR/$name.crt"
    chmod 0400 "$SERVICE_CERT_DIR/$name.key"
    chmod 0400 "$SERVICE_CERT_DIR/$name.pem"
}

grant_kafka_secret_group_read() {
    for path in "$@"; do
        chgrp "$KAFKA_SECRET_GROUP_ID" "$path"
        chmod 0440 "$path"
    done
}

mkdir -p "$SERVICE_CERT_DIR" "$SERVICE_CA_DIR"

wait_for_file "$VAULT_CACERT"
wait_for_file "$VAULT_TOKEN_FILE"
wait_for_file "$SERVICE_CA_BUNDLE_FILE"

VAULT_TOKEN="$(cat "$VAULT_TOKEN_FILE")"
export VAULT_TOKEN

until vault status >/dev/null 2>&1; do
    sleep 1
done

cp "$SERVICE_CA_BUNDLE_FILE" "$SERVICE_CERT_DIR/ca.crt"
cp "$SERVICE_CA_BUNDLE_FILE" "$SERVICE_CA_DIR/ca.crt"
chmod 0444 "$SERVICE_CERT_DIR/ca.crt" "$SERVICE_CA_DIR/ca.crt"

issue_cert "postgres" "postgres-server" "postgres" "identity-postgres-haproxy,access-control-postgres-haproxy,notification-postgres-haproxy,identity-postgresql,access-control-postgresql,notification-postgresql,localhost" "127.0.0.1"
issue_cert "redis" "redis-server" "redis" "identity-redis,access-control-redis,notification-redis,template-redis,localhost" "127.0.0.1"
issue_cert "kafka" "kafka-server" "kafka" "kafka,localhost" "127.0.0.1"
issue_cert "clickhouse" "clickhouse-server" "clickhouse" "clickhouse,localhost" "127.0.0.1"
issue_cert "client" "app-client" "monadiam-app-client" "" ""
grant_kafka_secret_group_read \
    "$SERVICE_CERT_DIR/kafka.key" \
    "$SERVICE_CERT_DIR/kafka.pem" \
    "$SERVICE_CERT_DIR/client.key" \
    "$SERVICE_CERT_DIR/client.pem"

echo "Service TLS material is available in $SERVICE_CERT_DIR."
