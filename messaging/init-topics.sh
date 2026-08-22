#!/bin/bash
set -euo pipefail

: "${KAFKA_CONNECT_CONFIG_STORAGE_TOPIC:?}"
: "${KAFKA_CONNECT_OFFSET_STORAGE_TOPIC:?}"
: "${KAFKA_CONNECT_STATUS_STORAGE_TOPIC:?}"
: "${KAFKA_CONNECT_GROUP_ID:?}"
: "${KAFKA_INTERNAL_PORT:?}"

SECRETS_DIR="${KAFKA_SECRETS_DIR:-/vault/secrets/kafka}"
BOOTSTRAP_SERVER="kafka:${KAFKA_INTERNAL_PORT}"
TLS_DIR="${KAFKA_TLS_DIR:-/tls}"
ADMIN_CONFIG="/tmp/kafka-admin.properties"

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

wait_for_file "$TLS_DIR/ca.crt"
wait_for_file "$TLS_DIR/client.pem"
wait_for_file "$SECRETS_DIR/kafka_admin_sasl_password"

ADMIN_PASSWORD="$(read_secret "$SECRETS_DIR/kafka_admin_sasl_password")"

cat > "$ADMIN_CONFIG" <<EOF
security.protocol=SASL_SSL
sasl.mechanism=SCRAM-SHA-512
sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="kafka-admin" password="${ADMIN_PASSWORD}";
ssl.keystore.type=PEM
ssl.keystore.location=${TLS_DIR}/client.pem
ssl.truststore.type=PEM
ssl.truststore.location=${TLS_DIR}/ca.crt
ssl.endpoint.identification.algorithm=https
EOF

chmod 0400 "$ADMIN_CONFIG"

KAFKA_TOPICS=(/opt/kafka/bin/kafka-topics.sh --bootstrap-server "$BOOTSTRAP_SERVER" --command-config "$ADMIN_CONFIG")
KAFKA_ACLS=(/opt/kafka/bin/kafka-acls.sh --bootstrap-server "$BOOTSTRAP_SERVER" --command-config "$ADMIN_CONFIG")

PARTITIONS=1
REPLICATION_FACTOR=1
RETENTION_MAIN=259200000
RETENTION_RETRY=604800000
RETENTION_DEAD=2592000000

TOPICS=()
topic_spec() {
    local spec="$1"
    local config
    shift
    for config in "$@"; do
        spec+="|$config"
    done
    TOPICS+=("$spec")
}

topic_triplet() {
    topic_spec "$1" "retention.ms=$RETENTION_MAIN"
    topic_spec "$1-retry" "retention.ms=$RETENTION_RETRY"
    topic_spec "$1-dead" "retention.ms=$RETENTION_DEAD"
}

for topic in realm account membership identifier service-client interface-client notification message-dispatch access-cache projection-journal; do
    topic_triplet "$topic"
done

topic_spec reauthentication "retention.ms=1800000"
topic_spec reauthentication-retry "retention.ms=$RETENTION_RETRY"
topic_spec reauthentication-dead "retention.ms=$RETENTION_DEAD"
topic_spec blacklist "cleanup.policy=compact,delete" "retention.ms=900000" "min.cleanable.dirty.ratio=0.01" "segment.ms=60000"
topic_spec blacklist-dead "retention.ms=$RETENTION_RETRY"
topic_spec audit-log-archive "retention.ms=$RETENTION_MAIN"
topic_spec change-log-archive "retention.ms=$RETENTION_MAIN"
topic_spec "$KAFKA_CONNECT_CONFIG_STORAGE_TOPIC" "cleanup.policy=compact"
topic_spec "$KAFKA_CONNECT_OFFSET_STORAGE_TOPIC" "cleanup.policy=compact"
topic_spec "$KAFKA_CONNECT_STATUS_STORAGE_TOPIC" "cleanup.policy=compact"

ALL_TOPICS=()
for spec in "${TOPICS[@]}"; do
    ALL_TOPICS+=("${spec%%|*}")
done

declare -A EXISTING_TOPIC_SET=()
while IFS= read -r topic; do
    [[ -z "$topic" ]] && continue
    EXISTING_TOPIC_SET["$topic"]=1
done < <("${KAFKA_TOPICS[@]}" --list)

create_topic() {
    local fields topic args i
    IFS='|' read -r -a fields <<< "$1"
    topic="${fields[0]}"
    [[ -n "${EXISTING_TOPIC_SET[$topic]+x}" ]] && return
    args=(--create --if-not-exists --topic "$topic" --partitions "$PARTITIONS" --replication-factor "$REPLICATION_FACTOR")
    for ((i = 1; i < ${#fields[@]}; i += 1)); do
        args+=(--config "${fields[$i]}")
    done
    "${KAFKA_TOPICS[@]}" "${args[@]}"
}

operation_args() {
    OPERATION_ARGS=()
    local operation
    IFS=',' read -r -a operations <<< "$1"
    for operation in "${operations[@]}"; do
        OPERATION_ARGS+=(--operation "$operation")
    done
}

add_acl() {
    local resource="$1"
    local principal="$2"
    local operations="$3"
    local value args
    shift 3
    operation_args "$operations"
    args=(--add --allow-principal "User:$principal" "${OPERATION_ARGS[@]}")
    case "$resource" in
        cluster) args+=(--cluster) ;;
        group) args+=(--group "$1") ;;
        topic) for value in "$@"; do args+=(--topic "$value"); done ;;
        *) echo "Unknown ACL resource: $resource" >&2; exit 1 ;;
    esac
    "${KAFKA_ACLS[@]}" "${args[@]}"
}

for spec in "${TOPICS[@]}"; do
    create_topic "$spec"
done

for principal in identity-service access-control-service notification-service kafka-connect clickhouse; do
    add_acl cluster "$principal" Describe
done
add_acl cluster kafka-ui Describe,DescribeConfigs

add_acl topic kafka-connect Read,Write,Describe "${ALL_TOPICS[@]}"
add_acl group kafka-connect Read,Describe "$KAFKA_CONNECT_GROUP_ID"

add_acl topic clickhouse Read,Describe audit-log-archive change-log-archive
add_acl group clickhouse Read,Describe clickhouse-audit-log
add_acl group clickhouse Read,Describe clickhouse-change-log

add_acl topic identity-service Read,Describe access-cache blacklist membership realm reauthentication access-cache-retry membership-retry realm-retry reauthentication-retry
add_acl topic identity-service Write,Describe access-cache access-cache-retry access-cache-dead blacklist-dead membership membership-retry membership-dead realm realm-retry realm-dead reauthentication reauthentication-retry reauthentication-dead
add_acl group identity-service Read,Describe identity-service-consumer-server
add_acl group identity-service Read,Describe identity-service-producer-client

add_acl topic access-control-service Read,Describe account blacklist membership projection-journal reauthentication account-retry membership-retry projection-journal-retry
add_acl topic access-control-service Write,Describe account account-retry account-dead blacklist-dead membership membership-retry membership-dead projection-journal projection-journal-retry projection-journal-dead reauthentication-retry reauthentication-dead
add_acl group access-control-service Read,Describe access-control-service-consumer-server
add_acl group access-control-service Read,Describe access-control-service-producer-client

add_acl topic notification-service Read,Describe access-cache blacklist reauthentication notification message-dispatch notification-retry message-dispatch-retry
add_acl topic notification-service Write,Describe access-cache access-cache-retry access-cache-dead blacklist-dead reauthentication reauthentication-retry reauthentication-dead notification notification-retry notification-dead message-dispatch message-dispatch-retry message-dispatch-dead
add_acl group notification-service Read,Describe notification-service-consumer-server
add_acl group notification-service Read,Describe notification-service-producer-client

add_acl topic kafka-ui Read,Describe,DescribeConfigs '*'
add_acl group kafka-ui Read,Describe '*'

echo "Kafka topics and ACLs are configured."
