#!/bin/sh
set -eu

: "${KAFKA_CONNECT_INTERNAL_PORT:?}"

CONNECT_URL="http://kafka-connect:${KAFKA_CONNECT_INTERNAL_PORT}"
CONNECTOR_TEMPLATE="/connector.outbox.json.tpl"

log() {
    printf '%s\n' "$*"
}

register_connector() {
    service_name="$1"
    connector_name="$2"
    postgres_db="$3"
    postgres_user="$4"
    postgres_password_file="$5"
    postgres_route_host="$6"
    postgres_route_port="$7"
    postgres_outbox_slot="$8"
    postgres_outbox_publication="$9"

    log "Waiting for ${service_name} PostgreSQL password..."
    i=0
    until [ -s "$postgres_password_file" ]; do
        i=$((i + 1))
        if [ "$i" -ge 60 ]; then
            echo "Timed out waiting for $postgres_password_file" >&2
            exit 1
        fi
        sleep 1
    done

    log "Registering Kafka Connect connector ${connector_name}..."

    export SERVICE_NAME="$service_name"
    export CONNECTOR_NAME="$connector_name"
    export POSTGRES_DB="$postgres_db"
    export POSTGRES_USER="$postgres_user"
    export POSTGRES_PASSWORD
    export POSTGRES_ROUTE_HOST="$postgres_route_host"
    export POSTGRES_ROUTE_PORT="$postgres_route_port"
    export POSTGRES_OUTBOX_SLOT="$postgres_outbox_slot"
    export POSTGRES_OUTBOX_PUBLICATION="$postgres_outbox_publication"

    POSTGRES_PASSWORD="$(cat "$postgres_password_file")"

    envsubst < "$CONNECTOR_TEMPLATE" \
      | curl -fsS -X PUT "${CONNECT_URL}/connectors/${connector_name}/config" \
        -H "Content-Type: application/json" \
        --data-binary @-

    log
    log "Registered Kafka Connect connector ${connector_name}."
}

log "Waiting for Kafka Connect at ${CONNECT_URL}..."
until curl -s -f "${CONNECT_URL}/" >/dev/null 2>&1; do
    sleep 2
done
log "Kafka Connect is reachable."

register_connector \
    "identity-service" \
    "identity-service-outbox-connector" \
    "${IDENTITY_POSTGRES_DB}" \
    "${IDENTITY_POSTGRES_USER}" \
    "/vault/secrets/identity/postgresql_password" \
    "identity-postgres-haproxy" \
    "${POSTGRES_HAPROXY_WRITE_PORT}" \
    "${IDENTITY_POSTGRES_OUTBOX_SLOT}" \
    "${IDENTITY_POSTGRES_OUTBOX_PUBLICATION}"

register_connector \
    "access-control-service" \
    "access-control-service-outbox-connector" \
    "${ACCESS_CONTROL_POSTGRES_DB}" \
    "${ACCESS_CONTROL_POSTGRES_USER}" \
    "/vault/secrets/access-control/postgresql_password" \
    "access-control-postgres-haproxy" \
    "${POSTGRES_HAPROXY_WRITE_PORT}" \
    "${ACCESS_CONTROL_POSTGRES_OUTBOX_SLOT}" \
    "${ACCESS_CONTROL_POSTGRES_OUTBOX_PUBLICATION}"

register_connector \
    "notification-service" \
    "notification-service-outbox-connector" \
    "${NOTIFICATION_POSTGRES_DB}" \
    "${NOTIFICATION_POSTGRES_USER}" \
    "/vault/secrets/notification/postgresql_password" \
    "notification-postgres-haproxy" \
    "${POSTGRES_HAPROXY_WRITE_PORT}" \
    "${NOTIFICATION_POSTGRES_OUTBOX_SLOT}" \
    "${NOTIFICATION_POSTGRES_OUTBOX_PUBLICATION}"

log "Registered connectors:"
curl -fsS "${CONNECT_URL}/connectors"
log
