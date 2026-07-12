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

# realm: access-control -> identity
$CREATE --topic realm       $RETENTION_MAIN
$CREATE --topic realm-retry $RETENTION_RETRY
$CREATE --topic realm-dead  $RETENTION_DEAD

# account: identity -> all services
$CREATE --topic account       $RETENTION_MAIN
$CREATE --topic account-retry $RETENTION_RETRY
$CREATE --topic account-dead  $RETENTION_DEAD

# membership: access-control -> identity
$CREATE --topic membership       $RETENTION_MAIN
$CREATE --topic membership-retry $RETENTION_RETRY
$CREATE --topic membership-dead  $RETENTION_DEAD

# identifier: identity -> notification
$CREATE --topic identifier       $RETENTION_MAIN
$CREATE --topic identifier-retry $RETENTION_RETRY
$CREATE --topic identifier-dead  $RETENTION_DEAD

# notification: all services -> notification
$CREATE --topic notification       $RETENTION_MAIN
$CREATE --topic notification-retry $RETENTION_RETRY
$CREATE --topic notification-dead  $RETENTION_DEAD

# access-cache: access-control -> all services
$CREATE --topic access-cache       $RETENTION_MAIN
$CREATE --topic access-cache-retry $RETENTION_RETRY
$CREATE --topic access-cache-dead  $RETENTION_DEAD

# projection-journal: access-control internal (redis sync)
$CREATE --topic projection-journal       $RETENTION_MAIN
$CREATE --topic projection-journal-retry $RETENTION_RETRY
$CREATE --topic projection-journal-dead  $RETENTION_DEAD

# reauthentication: identity -> all services
$CREATE --topic reauthentication       --config retention.ms=1800000 # 30 min
$CREATE --topic reauthentication-retry $RETENTION_RETRY
$CREATE --topic reauthentication-dead  $RETENTION_DEAD

# blacklist: session revocation
# compact + delete: deduplication by session key + TTL-eviction
$CREATE --topic blacklist \
  --config cleanup.policy=compact,delete \
  --config retention.ms=900000 \
  --config min.cleanable.dirty.ratio=0.01 \
  --config segment.ms=60000

# No retry required the data is temporary. Dead for failure analysis only.
$CREATE --topic blacklist-dead $RETENTION_RETRY

# audit-log-archive / change-log-archive: identity + access-control -> ClickHouse
# Pure fan-out consumption, no retries: if ClickHouse is down, messages just wait out the retention.
$CREATE --topic audit-log-archive  $RETENTION_MAIN
$CREATE --topic change-log-archive $RETENTION_MAIN
