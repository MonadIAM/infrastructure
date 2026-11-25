SELECT
    concat(object_key, '|', toString(row_count))
FROM
    audit.archive_log
WHERE
    table_name = {table:String}
AND
    partition = {partition:String}
ORDER BY
    archived_at DESC
LIMIT
    1
