#!/bin/sh
set -eu

INTERVAL="${ARCHIVE_INTERVAL_SECONDS:-86400}"

echo "clickhouse-archiver starting, will run every ${INTERVAL}s"

while true; do
    bash /archive.sh || echo "archive.sh failed, will retry next cycle" >&2
    sleep "$INTERVAL"
done
