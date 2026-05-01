#!/bin/bash
set -e

/opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:29092 --create --if-not-exists \
  --topic account \
  --partitions 1 \
  --replication-factor 1 \
  --config retention.ms=43200000

/opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:29092 --create --if-not-exists \
  --topic blacklist \
  --partitions 1 \
  --replication-factor 1 \
  --config cleanup.policy=compact,delete \
  --config retention.ms=900000 \
  --config min.cleanable.dirty.ratio=0.01 \
  --config segment.ms=100