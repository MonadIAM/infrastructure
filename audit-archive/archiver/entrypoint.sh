#!/bin/sh
set -eu

: "${ARCHIVE_INTERVAL_SECONDS:?}"
INTERVAL="$ARCHIVE_INTERVAL_SECONDS"

echo "clickhouse-archiver starting, will run every ${INTERVAL}s"

while true; do
    bash /archive.sh || echo "archive.sh failed, will retry next cycle" >&2
    sleep "$INTERVAL"
done
