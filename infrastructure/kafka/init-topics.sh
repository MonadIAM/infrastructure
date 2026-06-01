#!/bin/bash
set -e

KAFKA="/opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:29092"

# Tunables
PARTITIONS=1
REPLICATION_FACTOR=1

# Retention
RETENTION_MAIN="--config retention.ms=259200000"   # 3  days
RETENTION_RETRY="--config retention.ms=604800000"  # 7  days
RETENTION_DEAD="--config retention.ms=2592000000"  # 30 days
RETENTION_BLACKLIST="--config retention.ms=900000" # 15 min

# Common flags
CREATE="$KAFKA --create --if-not-exists --partitions $PARTITIONS --replication-factor $REPLICATION_FACTOR"

# account: identity -> all services
$CREATE --topic account       $RETENTION_MAIN
$CREATE --topic account-retry $RETENTION_RETRY
$CREATE --topic account-dead  $RETENTION_DEAD

# identifier: identity -> notification
$CREATE --topic identifier       $RETENTION_MAIN
$CREATE --topic identifier-retry $RETENTION_RETRY
$CREATE --topic identifier-dead  $RETENTION_DEAD

# notification: all services -> notification
$CREATE --topic notification       $RETENTION_MAIN
$CREATE --topic notification-retry $RETENTION_RETRY
$CREATE --topic notification-dead  $RETENTION_DEAD

# projection-journal: access-control internal (redis sync)
$CREATE --topic projection-journal       $RETENTION_MAIN
$CREATE --topic projection-journal-retry $RETENTION_RETRY
$CREATE --topic projection-journal-dead  $RETENTION_DEAD

# access-cache-invalidation: access-control -> all services
$CREATE --topic access-cache-invalidation       $RETENTION_MAIN
$CREATE --topic access-cache-invalidation-retry $RETENTION_RETRY
$CREATE --topic access-cache-invalidation-dead  $RETENTION_DEAD

# elevated-access: identity -> all services
$CREATE --topic elevated-access       --config retention.ms=1800000  # 30 min
$CREATE --topic elevated-access-retry $RETENTION_RETRY
$CREATE --topic elevated-access-dead  $RETENTION_DEAD

# blacklist: session revocation
# compact + delete: deduplication by session key + TTL-eviction
$CREATE --topic blacklist \
  --config cleanup.policy=compact,delete \
  --config retention.ms=900000 \
  --config min.cleanable.dirty.ratio=0.01 \
  --config segment.ms=60000

# No retry required the data is temporary. Dead for failure analysis only.
$CREATE --topic blacklist-dead $RETENTION_RETRY
