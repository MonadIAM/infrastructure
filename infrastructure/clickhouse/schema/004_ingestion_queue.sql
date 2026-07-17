CREATE TABLE IF NOT EXISTS audit.audit_log_queue
(
    actionType String,
    payload    String
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
    actionType String,
    payload    String
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
