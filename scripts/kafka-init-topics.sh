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
if [[ ! "$REALM_PARTITIONS" =~ ^[1-9][0-9]*$ ]]; then
    echo "REALM_PARTITIONS must be a positive integer" >&2
    exit 1
fi

if [[ ! "$POSITION_PARTITIONS" =~ ^[1-9][0-9]*$ ]]; then
    echo "POSITION_PARTITIONS must be a positive integer" >&2
    exit 1
fi

if [[ ! "$PROJECTION_JOURNAL_PARTITIONS" =~ ^[1-9][0-9]*$ ]]; then
    echo "PROJECTION_JOURNAL_PARTITIONS must be a positive integer" >&2
    exit 1
fi

topic_spec() {
    local spec="$1"
    local config
    shift
    for config in "$@"; do
        spec+="|$config"
    done
    TOPICS+=("$spec")
}

topic_pair() {
    topic_spec "$1" "retention.ms=$RETENTION_MAIN"
    topic_spec "$1-dead" "retention.ms=$RETENTION_DEAD"
}

for topic in realm position account identifier service-client interface-client notification message-dispatch access-cache projection-journal; do
    topic_pair "$topic"
done

topic_spec message-dispatch-retry-notification-service "retention.ms=$RETENTION_RETRY"

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
    local fields topic partitions args i
    IFS='|' read -r -a fields <<< "$1"
    topic="${fields[0]}"
    [[ -n "${EXISTING_TOPIC_SET[$topic]+x}" ]] && return
    partitions="$PARTITIONS"
    if [[ "$topic" == "projection-journal" ]]; then
        partitions="$PROJECTION_JOURNAL_PARTITIONS"
    elif [[ "$topic" == "realm" ]]; then
        partitions="$REALM_PARTITIONS"
    elif [[ "$topic" == "position" ]]; then
        partitions="$POSITION_PARTITIONS"
    fi
    args=(--create --if-not-exists --topic "$topic" --partitions "$partitions" --replication-factor "$REPLICATION_FACTOR")
    for ((i = 1; i < ${#fields[@]}; i += 1)); do
        args+=(--config "${fields[$i]}")
    done
    "${KAFKA_TOPICS[@]}" "${args[@]}"
}

for spec in "${TOPICS[@]}"; do
    create_topic "$spec"
done

realm_partitions=$("${KAFKA_TOPICS[@]}" --describe --topic realm | sed -n 's/.*PartitionCount: \([0-9]*\).*/\1/p' | head -n 1)
if [[ "$realm_partitions" != "$REALM_PARTITIONS" ]]; then
    echo "realm has $realm_partitions partitions; expected $REALM_PARTITIONS. Drain the ordered stream before changing its partition count." >&2
    exit 1
fi

position_partitions=$("${KAFKA_TOPICS[@]}" --describe --topic position | sed -n 's/.*PartitionCount: \([0-9]*\).*/\1/p' | head -n 1)
if [[ "$position_partitions" != "$POSITION_PARTITIONS" ]]; then
    echo "position has $position_partitions partitions; expected $POSITION_PARTITIONS. Drain the ordered stream before changing its partition count." >&2
    exit 1
fi

projection_partitions=$("${KAFKA_TOPICS[@]}" --describe --topic projection-journal | sed -n 's/.*PartitionCount: \([0-9]*\).*/\1/p' | head -n 1)
if [[ ! "$projection_partitions" =~ ^[0-9]+$ ]]; then
    echo "Unable to determine projection-journal partition count" >&2
    exit 1
fi
if ((projection_partitions < PROJECTION_JOURNAL_PARTITIONS)); then
    "${KAFKA_TOPICS[@]}" --alter --topic projection-journal --partitions "$PROJECTION_JOURNAL_PARTITIONS"
elif ((projection_partitions > PROJECTION_JOURNAL_PARTITIONS)); then
    echo "projection-journal has $projection_partitions partitions; expected $PROJECTION_JOURNAL_PARTITIONS" >&2
    exit 1
fi


echo "Kafka topics are configured."
