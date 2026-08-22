CREATE TABLE IF NOT EXISTS audit.archive_log
(
    table_name  LowCardinality(String),
    partition   String,
    row_count   UInt64,
    archived_at DateTime64(3, 'UTC')
)
ENGINE = MergeTree
ORDER BY (table_name, partition);

GRANT INSERT, SELECT ON audit.archive_log TO {{writer_user}};
GRANT SELECT ON audit.archive_log TO {{reader_user}};
