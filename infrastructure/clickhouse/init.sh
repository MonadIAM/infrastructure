#!/bin/bash
set -e

CLICKHOUSE_PASSWORD="$(cat /vault/secrets/clickhouse_password)"
CLICKHOUSE_WRITER_PASSWORD="$(cat /vault/secrets/clickhouse_writer_password)"
CLICKHOUSE_READER_PASSWORD="$(cat /vault/secrets/clickhouse_reader_password)"

CLICKHOUSE="clickhouse-client --host clickhouse --user ${CLICKHOUSE_USER} --password ${CLICKHOUSE_PASSWORD} --multiquery"

for file in /schema/*.sql; do
    echo "Applying $file"
    $CLICKHOUSE < "$file"
done

echo "Applying users/grants"
$CLICKHOUSE <<EOF
CREATE USER IF NOT EXISTS ${CLICKHOUSE_WRITER_USER} IDENTIFIED WITH sha256_password BY '${CLICKHOUSE_WRITER_PASSWORD}';
CREATE USER IF NOT EXISTS ${CLICKHOUSE_READER_USER} IDENTIFIED WITH sha256_password BY '${CLICKHOUSE_READER_PASSWORD}';

GRANT INSERT, SELECT ON audit.audit_log TO ${CLICKHOUSE_WRITER_USER};
GRANT INSERT, SELECT ON audit.change_log TO ${CLICKHOUSE_WRITER_USER};

GRANT SELECT ON audit.audit_log TO ${CLICKHOUSE_READER_USER};
GRANT SELECT ON audit.change_log TO ${CLICKHOUSE_READER_USER};
EOF
