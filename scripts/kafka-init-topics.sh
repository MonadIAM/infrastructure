#!/bin/bash
set -euo pipefail

BOOTSTRAP_SERVER="kafka:29092"
KAFKA_TOPICS=(/opt/kafka/bin/kafka-topics.sh --bootstrap-server "$BOOTSTRAP_SERVER")

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
    topic_spec "$1-dead" "retention.ms=$RETENTION_DEAD"
}

for topic in realm account membership identifier service-client interface-client notification message-dispatch access-cache projection-journal; do
    topic_triplet "$topic"
done

retry_topics() {
    local service="$1"
    local topic
    shift
    for topic in "$@"; do
        topic_spec "$topic-retry-$service" "retention.ms=$RETENTION_RETRY"
    done
}

retry_topics identity-service reauthentication access-cache membership realm
retry_topics access-control-service reauthentication account membership projection-journal
retry_topics notification-service reauthentication access-cache notification message-dispatch
retry_topics template-service reauthentication access-cache

topic_spec reauthentication "retention.ms=1800000"
topic_spec reauthentication-dead "retention.ms=$RETENTION_DEAD"
topic_spec blacklist "cleanup.policy=compact,delete" "retention.ms=900000" "min.cleanable.dirty.ratio=0.01" "segment.ms=60000"
topic_spec blacklist-dead "retention.ms=$RETENTION_RETRY"
topic_spec audit-log-archive "retention.ms=$RETENTION_MAIN"
topic_spec change-log-archive "retention.ms=$RETENTION_MAIN"
topic_spec "$KAFKA_CONNECT_CONFIG_STORAGE_TOPIC" "cleanup.policy=compact"
topic_spec "$KAFKA_CONNECT_OFFSET_STORAGE_TOPIC" "cleanup.policy=compact"
topic_spec "$KAFKA_CONNECT_STATUS_STORAGE_TOPIC" "cleanup.policy=compact"
topic_spec "${SCHEMA_REGISTRY_KAFKASQL_JOURNAL_TOPIC}" "cleanup.policy=delete" "retention.ms=-1" "retention.bytes=-1"
topic_spec "${SCHEMA_REGISTRY_KAFKASQL_SNAPSHOTS_TOPIC}" "cleanup.policy=delete" "retention.ms=-1" "retention.bytes=-1"
topic_spec "${SCHEMA_REGISTRY_EVENTS_TOPIC}" "cleanup.policy=delete" "retention.ms=-1" "retention.bytes=-1"

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

for spec in "${TOPICS[@]}"; do
    create_topic "$spec"
done

echo "Kafka topics are configured."
