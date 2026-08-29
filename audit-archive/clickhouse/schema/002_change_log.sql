CREATE DATABASE IF NOT EXISTS audit;

CREATE TABLE IF NOT EXISTS audit.change_log
(
    id          UUID,
    service     LowCardinality(String),
    created_at  DateTime64(3, 'UTC'),

    audit_entry UUID,

    change_type Enum8('create' = 1, 'update' = 2, 'delete' = 3, 'update_early' = 4, 'delete_early' = 5),
    entity_type LowCardinality(String),
    entity      UUID,

    delta       Nullable(String)
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(created_at)
ORDER BY (service, entity_type, created_at, id)
TTL toDateTime(created_at) + INTERVAL 180 DAY;
