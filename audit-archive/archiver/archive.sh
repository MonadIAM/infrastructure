#!/bin/bash
set -eu

: "${CLICKHOUSE_NATIVE_INTERNAL_PORT:?}"
: "${MINIO_API_INTERNAL_PORT:?}"

CLICKHOUSE_WRITER_PASSWORD="$(cat /vault/secrets/clickhouse/clickhouse_writer_password)"
MINIO_ARCHIVER_SECRET_KEY="$(cat /vault/secrets/minio/minio_archiver_secret_key)"

CH="clickhouse-client --host clickhouse --port ${CLICKHOUSE_NATIVE_INTERNAL_PORT} --secure --user ${CLICKHOUSE_WRITER_USER} --password ${CLICKHOUSE_WRITER_PASSWORD}"

archive_table() {
    table="$1"
    bucket="$2"

    cutoff=$($CH --query "SELECT toString(toYYYYMM(now() - INTERVAL ${ARCHIVE_AFTER_DAYS} DAY))")

    partitions=$($CH --query "
        SELECT DISTINCT partition FROM system.parts
        WHERE database = 'audit' AND table = '${table}' AND active AND partition <= '${cutoff}'
        ORDER BY partition
    ")

    for partition in $partitions; do
        already=$($CH --query "SELECT count() FROM audit.archive_log WHERE table_name = '${table}' AND partition = '${partition}'")
        if [ "$already" != "0" ]; then
            continue
        fi

        source_count=$($CH --query "
            SELECT sum(rows) FROM system.parts
            WHERE database = 'audit' AND table = '${table}' AND active AND partition = '${partition}'
        ")

        key="http://minio:${MINIO_API_INTERNAL_PORT}/${bucket}/${partition}.parquet"

        echo "Exporting ${table} partition ${partition} (${source_count} rows) -> ${bucket}/${partition}.parquet"
        $CH --query "
            INSERT INTO FUNCTION s3('${key}', '${MINIO_ARCHIVER_ACCESS_KEY}', '${MINIO_ARCHIVER_SECRET_KEY}', 'Parquet')
            SELECT * FROM audit.${table} WHERE toYYYYMM(created_at) = ${partition}
        "

        exported_count=$($CH --query "
            SELECT count() FROM s3('${key}', '${MINIO_ARCHIVER_ACCESS_KEY}', '${MINIO_ARCHIVER_SECRET_KEY}', 'Parquet')
        ")

        if [ "$exported_count" != "$source_count" ]; then
            echo "MISMATCH: ${table} partition ${partition} — source=${source_count} exported=${exported_count}, will retry next run" >&2
            continue
        fi

        $CH --query "
            INSERT INTO audit.archive_log (table_name, partition, row_count, archived_at)
            VALUES ('${table}', '${partition}', ${exported_count}, now64(3))
        "
        echo "Archived ${table} partition ${partition}: ${exported_count} rows"
    done
}

archive_table audit_log "${MINIO_AUDIT_LOG_BUCKET}"
archive_table change_log "${MINIO_CHANGE_LOG_BUCKET}"
