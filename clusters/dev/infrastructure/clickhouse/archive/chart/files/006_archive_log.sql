CREATE TABLE IF NOT EXISTS audit.archive_log (
    table_name  LowCardinality(String),
    partition   String,
    row_count   UInt64,
    object_key  String DEFAULT concat(partition, '.parquet'),
    archived_at DateTime64(3, 'UTC')
)
ENGINE = MergeTree
ORDER BY (
    table_name,
    partition
);

ALTER TABLE audit.archive_log
ADD COLUMN IF NOT EXISTS object_key String DEFAULT concat(partition, '.parquet');
