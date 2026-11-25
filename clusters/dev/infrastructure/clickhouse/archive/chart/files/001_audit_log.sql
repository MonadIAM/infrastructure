CREATE DATABASE IF NOT EXISTS audit;

CREATE TABLE IF NOT EXISTS audit.audit_log (
    id          UUID,
    service     LowCardinality(String),
    created_at  DateTime64(3, 'UTC'),
    ingested_at DateTime64(3, 'UTC') DEFAULT now64(3),

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
ORDER BY (
    service,
    entity_type,
    action_type,
    created_at,
    id
)
TTL toDateTime(ingested_at) + INTERVAL 180 DAY;

ALTER TABLE audit.audit_log
ADD COLUMN IF NOT EXISTS ingested_at DateTime64(3, 'UTC') DEFAULT now64(3);

ALTER TABLE audit.audit_log
MODIFY TTL toDateTime(ingested_at) + INTERVAL 180 DAY;
