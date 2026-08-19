#!/bin/sh
set -eu

CONNECT_URL="${CONNECT_URL:-http://kafka-connect:8083}"
CONNECTOR_TEMPLATE="/connector.outbox.json.tpl"

register_connector() {
    service_name="$1"
    connector_name="$2"
    postgres_db="$3"
    postgres_user="$4"
    postgres_password_file="$5"
    postgres_route_port="$6"
    postgres_outbox_slot="$7"
    postgres_outbox_publication="$8"

    i=0
    until [ -s "$postgres_password_file" ]; do
        i=$((i + 1))
        if [ "$i" -ge 60 ]; then
            echo "Timed out waiting for $postgres_password_file" >&2
            exit 1
        fi
        sleep 1
    done

    export SERVICE_NAME="$service_name"
    export CONNECTOR_NAME="$connector_name"
    export POSTGRES_DB="$postgres_db"
    export POSTGRES_USER="$postgres_user"
    export POSTGRES_PASSWORD
    export POSTGRES_ROUTE_PORT="$postgres_route_port"
    export POSTGRES_OUTBOX_SLOT="$postgres_outbox_slot"
    export POSTGRES_OUTBOX_PUBLICATION="$postgres_outbox_publication"

    POSTGRES_PASSWORD="$(cat "$postgres_password_file")"

    envsubst < "$CONNECTOR_TEMPLATE" \
      | curl -s -f -X PUT "${CONNECT_URL}/connectors/${connector_name}/config" \
        -H "Content-Type: application/json" \
        --data-binary @-
}

until curl -s -f "${CONNECT_URL}/" >/dev/null 2>&1; do
    echo "Waiting for Kafka Connect at ${CONNECT_URL}..."
    sleep 2
done

register_connector \
    "identity-service" \
    "identity-service-outbox-connector" \
    "${IDENTITY_POSTGRES_DB}" \
    "${IDENTITY_POSTGRES_USER}" \
    "/vault/secrets/identity/postgresql_password" \
    "${IDENTITY_POSTGRES_WRITE_PORT}" \
    "${IDENTITY_POSTGRES_OUTBOX_SLOT}" \
    "${IDENTITY_POSTGRES_OUTBOX_PUBLICATION}"

register_connector \
    "access-control-service" \
    "access-control-service-outbox-connector" \
    "${ACCESS_CONTROL_POSTGRES_DB}" \
    "${ACCESS_CONTROL_POSTGRES_USER}" \
    "/vault/secrets/access-control/postgresql_password" \
    "${ACCESS_CONTROL_POSTGRES_WRITE_PORT}" \
    "${ACCESS_CONTROL_POSTGRES_OUTBOX_SLOT}" \
    "${ACCESS_CONTROL_POSTGRES_OUTBOX_PUBLICATION}"

register_connector \
    "notification-service" \
    "notification-service-outbox-connector" \
    "${NOTIFICATION_POSTGRES_DB}" \
    "${NOTIFICATION_POSTGRES_USER}" \
    "/vault/secrets/notification/postgresql_password" \
    "${NOTIFICATION_POSTGRES_WRITE_PORT}" \
    "${NOTIFICATION_POSTGRES_OUTBOX_SLOT}" \
    "${NOTIFICATION_POSTGRES_OUTBOX_PUBLICATION}"

curl -s "${CONNECT_URL}/connectors"
