#!/bin/bash
set -eu

CH="clickhouse-client --host clickhouse --port 9000 --user ${CLICKHOUSE_WRITER_USER} --password ${CLICKHOUSE_WRITER_PASSWORD}"

archive_table() {
    table="$1"
    bucket="$2"

    case "$table" in
        audit_log)
            projection="* REPLACE (toString(id) AS id, toString(realm) AS realm, toString(actor) AS actor)"
            ;;
        change_log)
            projection="* REPLACE (toString(id) AS id, toString(audit_entry) AS audit_entry, toString(entity) AS entity)"
            ;;
    esac

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

        key="${SEAWEEDFS_ENDPOINT%/}/${bucket}/${partition}.parquet"

        echo "Exporting ${table} partition ${partition} (${source_count} rows) -> ${bucket}/${partition}.parquet"
        $CH --query "
            INSERT INTO FUNCTION s3('${key}', '${SEAWEEDFS_ARCHIVER_ACCESS_KEY}', '${SEAWEEDFS_ARCHIVER_SECRET_KEY}', 'Parquet')
            SELECT ${projection} FROM audit.${table} WHERE toYYYYMM(created_at) = ${partition}
        "

        exported_count=$($CH --query "
            SELECT count() FROM s3('${key}', '${SEAWEEDFS_ARCHIVER_ACCESS_KEY}', '${SEAWEEDFS_ARCHIVER_SECRET_KEY}', 'Parquet')
        ")

        if [ "$exported_count" != "$source_count" ]; then
            echo "MISMATCH: ${table} partition ${partition} - source=${source_count} exported=${exported_count}, will retry next run" >&2
            continue
        fi

        $CH --query "
            INSERT INTO audit.archive_log (table_name, partition, row_count, archived_at)
            VALUES ('${table}', '${partition}', ${exported_count}, now64(3))
        "
        echo "Archived ${table} partition ${partition}: ${exported_count} rows"
    done
}

archive_table audit_log "${SEAWEEDFS_AUDIT_LOG_BUCKET}"
archive_table change_log "${SEAWEEDFS_CHANGE_LOG_BUCKET}"
