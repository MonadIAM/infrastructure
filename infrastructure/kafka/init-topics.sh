#!/bin/bash
set -e

KAFKA="/opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:29092"
RETENTION_MAIN="--config retention.ms=259200000"   # 3 days
RETENTION_DLQ="--config retention.ms=604800000"    # 7 days

# projection-journal: access-control internal projection redis synchronization
$KAFKA --create --if-not-exists --topic projection-journal \
  --partitions 1 --replication-factor 1 $RETENTION_MAIN

$KAFKA --create --if-not-exists --topic projection-journal-dlq \
  --partitions 1 --replication-factor 1 $RETENTION_DLQ

# access-cache-invalidation: access-control -> services
$KAFKA --create --if-not-exists --topic access-cache-invalidation \
  --partitions 1 --replication-factor 1 $RETENTION_MAIN

$KAFKA --create --if-not-exists --topic access-cache-invalidation-dlq \
  --partitions 1 --replication-factor 1 $RETENTION_DLQ

# access-cache-snapshot: access-control -> services
$KAFKA --create --if-not-exists --topic access-cache-snapshot \
  --partitions 1 --replication-factor 1 $RETENTION_MAIN

$KAFKA --create --if-not-exists --topic access-cache-snapshot-dlq \
  --partitions 1 --replication-factor 1 $RETENTION_DLQ

# identifier: identity -> notification
$KAFKA --create --if-not-exists --topic identifier \
  --partitions 1 --replication-factor 1 $RETENTION_MAIN

$KAFKA --create --if-not-exists --topic identifier-dlq \
  --partitions 1 --replication-factor 1 $RETENTION_DLQ

# notification: all other services -> notification
$KAFKA --create --if-not-exists --topic notification \
  --partitions 1 --replication-factor 1 $RETENTION_MAIN

$KAFKA --create --if-not-exists --topic notification-dlq \
  --partitions 1 --replication-factor 1 $RETENTION_DLQ

# account: identity -> all other services
$KAFKA --create --if-not-exists --topic account \
  --partitions 1 --replication-factor 1 $RETENTION_MAIN

$KAFKA --create --if-not-exists --topic account-dlq \
  --partitions 1 --replication-factor 1 $RETENTION_DLQ

# blacklist: access token invalidation
$KAFKA --create --if-not-exists --topic blacklist \
  --partitions 1 --replication-factor 1 \
  --config cleanup.policy=compact,delete \
  --config retention.ms=900000 \
  --config min.cleanable.dirty.ratio=0.01 \
  --config segment.ms=100
