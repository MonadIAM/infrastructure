CREATE DATABASE IF NOT EXISTS audit;

CREATE TABLE IF NOT EXISTS audit.audit_log
(
    id          UUID,
    service     LowCardinality(String),
    created_at  DateTime64(3, 'UTC'),

    action_type LowCardinality(String),
    entity_type LowCardinality(String),

    realm       Nullable(UUID),
    actor       Nullable(UUID),

    ip          Nullable(String),
    user_agent  Nullable(String),

    input       Nullable(String)
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(created_at)
ORDER BY (service, entity_type, action_type, created_at, id)
TTL toDateTime(created_at) + INTERVAL 180 DAY;
