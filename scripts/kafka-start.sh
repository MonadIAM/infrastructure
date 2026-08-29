#!/bin/bash
set -euo pipefail

KAFKA_CONFIG_FILE="/etc/kafka/server.properties"
KAFKA_DATA_DIR="/var/lib/kafka/data"

mkdir -p "$KAFKA_DATA_DIR"

if [ ! -f "$KAFKA_DATA_DIR/meta.properties" ]; then
    /opt/kafka/bin/kafka-storage.sh format \
        --ignore-formatted \
        --cluster-id "$KAFKA_CLUSTER_ID" \
        --config "$KAFKA_CONFIG_FILE"
else
    echo "Kafka storage is already formatted."
fi

exec /opt/kafka/bin/kafka-server-start.sh "$KAFKA_CONFIG_FILE"
