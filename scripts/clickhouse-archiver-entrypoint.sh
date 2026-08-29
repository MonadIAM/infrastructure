#!/bin/sh
set -eu

echo "clickhouse-archiver starting, will run every ${ARCHIVE_INTERVAL_SECONDS}s"

while true; do
    bash /scripts/clickhouse-archive.sh || echo "clickhouse-archive.sh failed, will retry next cycle" >&2
    sleep "$ARCHIVE_INTERVAL_SECONDS"
done
