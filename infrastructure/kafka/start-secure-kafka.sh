#!/bin/bash
set -euo pipefail

: "${CLUSTER_ID:?}"
: "${KAFKA_INTERNAL_PORT:?}"
: "${KAFKA_CONTROLLER_PORT:?}"
: "${KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR:?}"
: "${KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR:?}"
: "${KAFKA_TRANSACTION_STATE_LOG_MIN_ISR:?}"

KAFKA_CONFIG_FILE="${KAFKA_CONFIG_FILE:-/home/appuser/kafka/server.properties}"
KAFKA_HEALTH_CONFIG_FILE="${KAFKA_HEALTH_CONFIG_FILE:-/home/appuser/kafka/health.properties}"
KAFKA_SECRETS_DIR="${KAFKA_SECRETS_DIR:-/vault/secrets/kafka}"
KAFKA_TLS_DIR="${KAFKA_TLS_DIR:-/tls}"
KAFKA_DATA_DIR="${KAFKA_LOG_DIRS:-/var/lib/kafka/data}"

read_secret() {
    local path="$1"
    tr -d '\r\n' < "$path"
}

wait_for_file() {
    local path="$1"
    local i=0

    until [ -s "$path" ]; do
        i=$((i + 1))
        if [ "$i" -ge 60 ]; then
            echo "Timed out waiting for $path" >&2
            exit 1
        fi
        sleep 1
    done
}

wait_for_file "$KAFKA_TLS_DIR/ca.crt"
wait_for_file "$KAFKA_TLS_DIR/kafka.pem"
wait_for_file "$KAFKA_TLS_DIR/client.pem"
wait_for_file "$KAFKA_SECRETS_DIR/kafka_broker_sasl_password"
wait_for_file "$KAFKA_SECRETS_DIR/kafka_admin_sasl_password"
wait_for_file "$KAFKA_SECRETS_DIR/kafka_connect_sasl_password"
wait_for_file "$KAFKA_SECRETS_DIR/kafka_ui_sasl_password"
wait_for_file "$KAFKA_SECRETS_DIR/kafka_clickhouse_sasl_password"
wait_for_file "$KAFKA_SECRETS_DIR/identity/kafka_sasl_password"
wait_for_file "$KAFKA_SECRETS_DIR/access-control/kafka_sasl_password"
wait_for_file "$KAFKA_SECRETS_DIR/notification/kafka_sasl_password"

BROKER_PASSWORD="$(read_secret "$KAFKA_SECRETS_DIR/kafka_broker_sasl_password")"
ADMIN_PASSWORD="$(read_secret "$KAFKA_SECRETS_DIR/kafka_admin_sasl_password")"
CONNECT_PASSWORD="$(read_secret "$KAFKA_SECRETS_DIR/kafka_connect_sasl_password")"
UI_PASSWORD="$(read_secret "$KAFKA_SECRETS_DIR/kafka_ui_sasl_password")"
CLICKHOUSE_PASSWORD="$(read_secret "$KAFKA_SECRETS_DIR/kafka_clickhouse_sasl_password")"
IDENTITY_PASSWORD="$(read_secret "$KAFKA_SECRETS_DIR/identity/kafka_sasl_password")"
ACCESS_CONTROL_PASSWORD="$(read_secret "$KAFKA_SECRETS_DIR/access-control/kafka_sasl_password")"
NOTIFICATION_PASSWORD="$(read_secret "$KAFKA_SECRETS_DIR/notification/kafka_sasl_password")"

mkdir -p "$(dirname "$KAFKA_CONFIG_FILE")" "$KAFKA_DATA_DIR"
rm -f "$KAFKA_CONFIG_FILE" "$KAFKA_HEALTH_CONFIG_FILE"

cat > "$KAFKA_CONFIG_FILE" <<EOF
node.id=1
process.roles=broker,controller
controller.quorum.voters=1@kafka:${KAFKA_CONTROLLER_PORT}
controller.listener.names=CONTROLLER
listeners=BROKER://:${KAFKA_INTERNAL_PORT},CONTROLLER://:${KAFKA_CONTROLLER_PORT}
advertised.listeners=BROKER://kafka:${KAFKA_INTERNAL_PORT}
listener.security.protocol.map=BROKER:SASL_SSL,CONTROLLER:SASL_SSL
inter.broker.listener.name=BROKER

log.dirs=${KAFKA_DATA_DIR}
num.partitions=1
auto.create.topics.enable=false
offsets.topic.replication.factor=${KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR}
transaction.state.log.replication.factor=${KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR}
transaction.state.log.min.isr=${KAFKA_TRANSACTION_STATE_LOG_MIN_ISR}

sasl.enabled.mechanisms=SCRAM-SHA-512
sasl.mechanism.inter.broker.protocol=SCRAM-SHA-512
sasl.mechanism.controller.protocol=SCRAM-SHA-512
listener.name.broker.sasl.enabled.mechanisms=SCRAM-SHA-512
listener.name.controller.sasl.enabled.mechanisms=SCRAM-SHA-512
listener.name.broker.scram-sha-512.sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="kafka-broker" password="${BROKER_PASSWORD}";
listener.name.controller.scram-sha-512.sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="kafka-broker" password="${BROKER_PASSWORD}";

ssl.keystore.type=PEM
ssl.keystore.location=${KAFKA_TLS_DIR}/kafka.pem
ssl.truststore.type=PEM
ssl.truststore.location=${KAFKA_TLS_DIR}/ca.crt
ssl.client.auth=required
ssl.enabled.protocols=TLSv1.3,TLSv1.2
ssl.protocol=TLSv1.3
ssl.cipher.suites=TLS_AES_256_GCM_SHA384,TLS_AES_128_GCM_SHA256,TLS_CHACHA20_POLY1305_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
ssl.endpoint.identification.algorithm=https

authorizer.class.name=org.apache.kafka.metadata.authorizer.StandardAuthorizer
super.users=User:kafka-admin;User:kafka-broker
allow.everyone.if.no.acl.found=false
EOF

chmod 0400 "$KAFKA_CONFIG_FILE"

cat > "$KAFKA_HEALTH_CONFIG_FILE" <<EOF
security.protocol=SASL_SSL
sasl.mechanism=SCRAM-SHA-512
sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="kafka-admin" password="${ADMIN_PASSWORD}";
ssl.keystore.type=PEM
ssl.keystore.location=${KAFKA_TLS_DIR}/client.pem
ssl.truststore.type=PEM
ssl.truststore.location=${KAFKA_TLS_DIR}/ca.crt
ssl.endpoint.identification.algorithm=https
EOF

chmod 0400 "$KAFKA_CONFIG_FILE" "$KAFKA_HEALTH_CONFIG_FILE"

if [ ! -f "$KAFKA_DATA_DIR/meta.properties" ]; then
    /opt/kafka/bin/kafka-storage.sh format \
        --ignore-formatted \
        --cluster-id "$CLUSTER_ID" \
        --config "$KAFKA_CONFIG_FILE" \
        --add-scram "SCRAM-SHA-512=[name=kafka-broker,password=${BROKER_PASSWORD}]" \
        --add-scram "SCRAM-SHA-512=[name=kafka-admin,password=${ADMIN_PASSWORD}]" \
        --add-scram "SCRAM-SHA-512=[name=kafka-connect,password=${CONNECT_PASSWORD}]" \
        --add-scram "SCRAM-SHA-512=[name=kafka-ui,password=${UI_PASSWORD}]" \
        --add-scram "SCRAM-SHA-512=[name=clickhouse,password=${CLICKHOUSE_PASSWORD}]" \
        --add-scram "SCRAM-SHA-512=[name=identity-service,password=${IDENTITY_PASSWORD}]" \
        --add-scram "SCRAM-SHA-512=[name=access-control-service,password=${ACCESS_CONTROL_PASSWORD}]" \
        --add-scram "SCRAM-SHA-512=[name=notification-service,password=${NOTIFICATION_PASSWORD}]"
else
    echo "Kafka storage is already formatted; keeping existing SCRAM metadata."
fi

exec /opt/kafka/bin/kafka-server-start.sh "$KAFKA_CONFIG_FILE"
