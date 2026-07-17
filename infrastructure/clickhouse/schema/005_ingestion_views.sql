CREATE MATERIALIZED VIEW IF NOT EXISTS audit.audit_log_mv TO audit.audit_log
DEFINER = {{writer_user}} SQL SECURITY DEFINER
AS SELECT
    JSONExtractString(payload, 'id')                                        AS id,
    JSONExtractString(payload, 'service')                                   AS service,
    parseDateTime64BestEffort(JSONExtractString(payload, 'created_at'), 3)  AS created_at,
    JSONExtractString(payload, 'action_type')                               AS action_type,
    JSONExtractString(payload, 'entity_type')                               AS entity_type,
    NULLIF(JSONExtractString(payload, 'realm'), '')                         AS realm,
    NULLIF(JSONExtractString(payload, 'actor'), '')                         AS actor,
    NULLIF(JSONExtractString(payload, 'ip'), '')                            AS ip,
    NULLIF(JSONExtractString(payload, 'user_agent'), '')                    AS user_agent,
    NULLIF(JSONExtractString(payload, 'input'), '')                         AS input
FROM audit.audit_log_queue;

CREATE MATERIALIZED VIEW IF NOT EXISTS audit.change_log_mv TO audit.change_log
DEFINER = {{writer_user}} SQL SECURITY DEFINER
AS SELECT
    JSONExtractString(payload, 'id')                                        AS id,
    JSONExtractString(payload, 'service')                                   AS service,
    parseDateTime64BestEffort(JSONExtractString(payload, 'created_at'), 3)  AS created_at,
    JSONExtractString(payload, 'audit_entry')                               AS audit_entry,
    JSONExtractString(payload, 'change_type')                               AS change_type,
    JSONExtractString(payload, 'entity_type')                               AS entity_type,
    JSONExtractString(payload, 'entity')                                    AS entity,
    NULLIF(JSONExtractString(payload, 'delta'), '')                         AS delta
FROM audit.change_log_queue;
