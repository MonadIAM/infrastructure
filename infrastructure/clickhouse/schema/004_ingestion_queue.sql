CREATE TABLE IF NOT EXISTS audit.audit_log_queue
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
ENGINE = Kafka
SETTINGS
    kafka_broker_list = 'kafka:29092',
    kafka_topic_list = 'audit-log-archive',
    kafka_group_name = 'clickhouse-audit-log',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1;

CREATE TABLE IF NOT EXISTS audit.change_log_queue
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
ENGINE = Kafka
SETTINGS
    kafka_broker_list = 'kafka:29092',
    kafka_topic_list = 'change-log-archive',
    kafka_group_name = 'clickhouse-change-log',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1;

GRANT SELECT ON audit.audit_log_queue TO {{writer_user}};
GRANT SELECT ON audit.change_log_queue TO {{writer_user}};
