CREATE TABLE IF NOT EXISTS audit.audit_log_queue (
    actionType String,
    payload    String
)
ENGINE = Kafka(archive_kafka)
SETTINGS
    kafka_broker_list = 'kafka-main-kafka-bootstrap.monadiam-messaging.svc:9093',
    kafka_topic_list = 'audit-log-archive',
    kafka_group_name = 'clickhouse-audit-log',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1;

CREATE TABLE IF NOT EXISTS audit.change_log_queue (
    actionType String,
    payload    String
)
ENGINE = Kafka(archive_kafka)
SETTINGS
    kafka_broker_list = 'kafka-main-kafka-bootstrap.monadiam-messaging.svc:9093',
    kafka_topic_list = 'change-log-archive',
    kafka_group_name = 'clickhouse-change-log',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1;
