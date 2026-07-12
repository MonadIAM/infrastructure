CREATE MATERIALIZED VIEW IF NOT EXISTS audit.audit_log_mv TO audit.audit_log
DEFINER = {{writer_user}} SQL SECURITY DEFINER
AS SELECT * FROM audit.audit_log_queue;

CREATE MATERIALIZED VIEW IF NOT EXISTS audit.change_log_mv TO audit.change_log
DEFINER = {{writer_user}} SQL SECURITY DEFINER
AS SELECT * FROM audit.change_log_queue;
