INSERT INTO audit.archive_log (
    table_name,
    partition,
    row_count,
    object_key,
    archived_at
)
SELECT
    {table:String},
    {partition:String},
    {row_count:UInt64},
    {object_key:String},
    now64(3)
