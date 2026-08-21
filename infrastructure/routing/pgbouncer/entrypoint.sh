#!/bin/sh
set -eu

: "${IDENTITY_POSTGRES_DB:?}"
: "${IDENTITY_POSTGRES_USER:?}"
: "${IDENTITY_POSTGRES_TARGET_PORT:?}"
: "${ACCESS_CONTROL_POSTGRES_DB:?}"
: "${ACCESS_CONTROL_POSTGRES_USER:?}"
: "${ACCESS_CONTROL_POSTGRES_TARGET_PORT:?}"
: "${NOTIFICATION_POSTGRES_DB:?}"
: "${NOTIFICATION_POSTGRES_USER:?}"
: "${NOTIFICATION_POSTGRES_TARGET_PORT:?}"
: "${PGBOUNCER_LISTEN_PORT:?}"

wait_for_secret() {
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

wait_for_secret /secrets/identity/postgresql_password
wait_for_secret /secrets/access-control/postgresql_password
wait_for_secret /secrets/notification/postgresql_password

TLS_DIR=/tmp/pgbouncer-tls
mkdir -p "$TLS_DIR"
cp /tls/ca.crt "$TLS_DIR/ca.crt"
cp /tls/pgbouncer.crt "$TLS_DIR/pgbouncer.crt"
cp /tls/pgbouncer.key "$TLS_DIR/pgbouncer.key"
cp /tls/client.crt "$TLS_DIR/client.crt"
cp /tls/client.key "$TLS_DIR/client.key"
chown -R pgbouncer:pgbouncer "$TLS_DIR"
chmod 0444 "$TLS_DIR/ca.crt" "$TLS_DIR/pgbouncer.crt" "$TLS_DIR/client.crt"
chmod 0400 "$TLS_DIR/pgbouncer.key" "$TLS_DIR/client.key"

envsubst '${IDENTITY_POSTGRES_DB} ${IDENTITY_POSTGRES_USER} ${IDENTITY_POSTGRES_TARGET_PORT} ${ACCESS_CONTROL_POSTGRES_DB} ${ACCESS_CONTROL_POSTGRES_USER} ${ACCESS_CONTROL_POSTGRES_TARGET_PORT} ${NOTIFICATION_POSTGRES_DB} ${NOTIFICATION_POSTGRES_USER} ${NOTIFICATION_POSTGRES_TARGET_PORT} ${PGBOUNCER_LISTEN_PORT} ${PGBOUNCER_MAX_CLIENT_CONN} ${PGBOUNCER_DEFAULT_POOL_SIZE} ${PGBOUNCER_RESERVE_POOL_SIZE}' \
    < /etc/pgbouncer/pgbouncer.ini.tpl > /tmp/pgbouncer.ini

{
    printf '"%s" "%s"\n' "${IDENTITY_POSTGRES_USER}" "$(cat /secrets/identity/postgresql_password)"
    printf '"%s" "%s"\n' "${ACCESS_CONTROL_POSTGRES_USER}" "$(cat /secrets/access-control/postgresql_password)"
    printf '"%s" "%s"\n' "${NOTIFICATION_POSTGRES_USER}" "$(cat /secrets/notification/postgresql_password)"
} > /tmp/userlist.txt

exec pgbouncer /tmp/pgbouncer.ini
