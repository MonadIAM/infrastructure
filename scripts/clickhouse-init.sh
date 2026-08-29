#!/bin/bash
set -e

CLICKHOUSE="clickhouse-client --host clickhouse --port 9000 --user ${CLICKHOUSE_USER} --password ${CLICKHOUSE_PASSWORD} --multiquery"

for file in /schema/*.sql; do
    echo "Applying $file"
    sed \
        -e "s|{{writer_user}}|${CLICKHOUSE_WRITER_USER}|g" \
        -e "s|{{writer_password}}|${CLICKHOUSE_WRITER_PASSWORD}|g" \
        -e "s|{{reader_user}}|${CLICKHOUSE_READER_USER}|g" \
        -e "s|{{reader_password}}|${CLICKHOUSE_READER_PASSWORD}|g" \
        "$file" | $CLICKHOUSE
done
