#!/bin/bash
set -eu

: "${PATRONI_SCOPE:?PATRONI_SCOPE must be set per cluster}"
: "${POSTGRES_DB:?POSTGRES_DB must be set per cluster}"
: "${POSTGRES_USER:?POSTGRES_USER must be set per cluster}"
: "${POSTGRES_REPLICATION_USER:?POSTGRES_REPLICATION_USER must be set per cluster}"
: "${POSTGRES_OUTBOX_SLOT:?POSTGRES_OUTBOX_SLOT must be set per cluster}"

export PATRONI_NAME="${PATRONI_NAME:-$(hostname)}"
PATRONI_NODE_HOST="${PATRONI_NODE_HOST:-$(hostname -i | awk '{print $1}')}"

export PATRONI_SUPERUSER_USERNAME="${POSTGRES_USER}"
export PATRONI_SUPERUSER_PASSWORD="$(cat /vault/secrets/postgresql_password)"
export PATRONI_REPLICATION_USERNAME="${POSTGRES_REPLICATION_USER}"
export PATRONI_REPLICATION_PASSWORD="$(cat /vault/secrets/postgresql_replication_password)"

export PATRONI_POSTGRESQL_CONNECT_ADDRESS="${PATRONI_NODE_HOST}:5432"
export PATRONI_RESTAPI_CONNECT_ADDRESS="${PATRONI_NODE_HOST}:8008"

envsubst < /patroni.yml.tpl > /tmp/patroni.yml

PGDATA_ROOT="/var/lib/postgresql/data"
mkdir -p "$PGDATA_ROOT"
chown -R postgres:postgres "$PGDATA_ROOT"
chmod 700 "$PGDATA_ROOT"

exec gosu postgres patroni /tmp/patroni.yml
