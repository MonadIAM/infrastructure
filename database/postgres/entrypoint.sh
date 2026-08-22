#!/bin/bash
set -eu

: "${PATRONI_SCOPE:?}"
: "${POSTGRES_DB:?}"
: "${POSTGRES_USER:?}"
: "${POSTGRES_REPLICATION_USER:?}"
: "${POSTGRES_OUTBOX_SLOT:?}"

if [ -n "${PATRONI_NAME+x}" ]; then
    : "${PATRONI_NAME:?}"
else
    : "${HOSTNAME:?}"
    PATRONI_NAME="$HOSTNAME"
fi
export PATRONI_NAME

if [ -n "${PATRONI_NODE_HOST+x}" ]; then
    : "${PATRONI_NODE_HOST:?}"
else
    PATRONI_NODE_HOST="$(hostname -i | awk '{print $1}')"
    : "${PATRONI_NODE_HOST:?}"
fi

export PATRONI_SUPERUSER_USERNAME="${POSTGRES_USER}"
export PATRONI_SUPERUSER_PASSWORD="$(cat /vault/secrets/postgresql_password)"
export PATRONI_REPLICATION_USERNAME="${POSTGRES_REPLICATION_USER}"
export PATRONI_REPLICATION_PASSWORD="$(cat /vault/secrets/postgresql_replication_password)"

export PATRONI_POSTGRESQL_CONNECT_ADDRESS="${PATRONI_NODE_HOST}:5432"
export PATRONI_RESTAPI_CONNECT_ADDRESS="${PATRONI_NODE_HOST}:8008"

TLS_DIR=/tmp/postgres-tls
install -d -m 0700 -o postgres -g postgres "$TLS_DIR"
cp /tls/ca.crt "$TLS_DIR/ca.crt"
cp /tls/postgres.crt "$TLS_DIR/postgres.crt"
cp /tls/postgres.key "$TLS_DIR/postgres.key"
chown postgres:postgres "$TLS_DIR/ca.crt" "$TLS_DIR/postgres.crt" "$TLS_DIR/postgres.key"
chmod 0444 "$TLS_DIR/ca.crt" "$TLS_DIR/postgres.crt"
chmod 0400 "$TLS_DIR/postgres.key"

export POSTGRES_SSL_CA_FILE="$TLS_DIR/ca.crt"
export POSTGRES_SSL_CERT_FILE="$TLS_DIR/postgres.crt"
export POSTGRES_SSL_KEY_FILE="$TLS_DIR/postgres.key"

envsubst < /patroni.yml.tpl > /tmp/patroni.yml

PGDATA_ROOT="/var/lib/postgresql/data"
mkdir -p "$PGDATA_ROOT"
chown -R postgres:postgres "$PGDATA_ROOT"
chmod 700 "$PGDATA_ROOT"

exec gosu postgres patroni /tmp/patroni.yml
