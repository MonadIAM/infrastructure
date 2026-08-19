{
  "name": "${CONNECTOR_NAME}",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "database.hostname": "postgres-haproxy",
    "database.port": "${POSTGRES_ROUTE_PORT}",
    "database.user": "${POSTGRES_USER}",
    "database.password": "${POSTGRES_PASSWORD}",
    "database.dbname": "${POSTGRES_DB}",
    "topic.prefix": "${SERVICE_NAME}",
    "schema.include.list": "system",
    "table.include.list": "system\\.outbox",
    "plugin.name": "pgoutput",
    "slot.name": "${POSTGRES_OUTBOX_SLOT}",
    "publication.name": "${POSTGRES_OUTBOX_PUBLICATION}",
    "publication.autocreate.mode": "filtered",
    "snapshot.mode": "no_data",

    "value.converter": "org.apache.kafka.connect.json.JsonConverter",
    "value.converter.schemas.enable": "false",
    "key.converter": "org.apache.kafka.connect.json.JsonConverter",
    "key.converter.schemas.enable": "false",

    "transforms": "outbox",
    "transforms.outbox.type": "io.debezium.transforms.outbox.EventRouter",
    "transforms.outbox.route.topic.replacement": "ROUTEDBYVALUE",
    "transforms.outbox.route.by.field": "destination_topic",
    "transforms.outbox.table.field.event.key": "id",
    "transforms.outbox.table.field.event.type": "action_type",
    "transforms.outbox.table.expand.json.payload": "true",
    "transforms.outbox.table.fields.additional.placement": "action_type:envelope:actionType"
  }
}
