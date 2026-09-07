#!/bin/sh
set -eu

CONNECT_URL="http://kafka-connect:8083"
CONNECTOR_TEMPLATE="/debezium/outbox.connector.json"

log() {
    printf '%s\n' "$*"
}

register_connector() {
    log "Registering Kafka Connect connector $2..."

    SERVICE_NAME="$1" \
    POSTGRES_DB="$3" \
    POSTGRES_USER="$4" \
    POSTGRES_PASSWORD="$5" \
    POSTGRES_ROUTE_HOST="$6" \
    POSTGRES_ROUTE_PORT="$7" \
    POSTGRES_OUTBOX_SLOT="$8" \
    POSTGRES_OUTBOX_PUBLICATION="$9" \
    routedByValue='${routedByValue}' \
    envsubst < "$CONNECTOR_TEMPLATE" \
      | curl -fsS -X PUT "${CONNECT_URL}/connectors/$2/config" \
        -H "Content-Type: application/json" \
        --data-binary @-

    log
    log "Registered Kafka Connect connector $2."
}

log "Waiting for Kafka Connect at ${CONNECT_URL}..."
until curl -s -f "${CONNECT_URL}/" >/dev/null 2>&1; do
    sleep 2
done
log "Kafka Connect is reachable."

register_connector identity-service identity-service-outbox-connector \
    "$IDENTITY_POSTGRES_DB" "$IDENTITY_POSTGRES_USER" "$IDENTITY_POSTGRES_PASSWORD" \
    identity-postgresql 5432 "$IDENTITY_POSTGRES_OUTBOX_SLOT" "$IDENTITY_POSTGRES_OUTBOX_PUBLICATION"

register_connector template-service template-service-outbox-connector \
    "$TEMPLATE_POSTGRES_DB" "$TEMPLATE_POSTGRES_USER" "$TEMPLATE_POSTGRES_PASSWORD" \
    template-postgresql 5432 "$TEMPLATE_POSTGRES_OUTBOX_SLOT" "$TEMPLATE_POSTGRES_OUTBOX_PUBLICATION"

register_connector access-control-service access-control-service-outbox-connector \
    "$ACCESS_CONTROL_POSTGRES_DB" "$ACCESS_CONTROL_POSTGRES_USER" "$ACCESS_CONTROL_POSTGRES_PASSWORD" \
    access-control-postgresql 5432 "$ACCESS_CONTROL_POSTGRES_OUTBOX_SLOT" "$ACCESS_CONTROL_POSTGRES_OUTBOX_PUBLICATION"

register_connector notification-service notification-service-outbox-connector \
    "$NOTIFICATION_POSTGRES_DB" "$NOTIFICATION_POSTGRES_USER" "$NOTIFICATION_POSTGRES_PASSWORD" \
    notification-postgresql 5432 "$NOTIFICATION_POSTGRES_OUTBOX_SLOT" "$NOTIFICATION_POSTGRES_OUTBOX_PUBLICATION"

log "Registered connectors:"
curl -fsS "${CONNECT_URL}/connectors"
log
