CREATE USER IF NOT EXISTS {{writer_user}} IDENTIFIED WITH sha256_password BY '{{writer_password}}';
CREATE USER IF NOT EXISTS {{reader_user}} IDENTIFIED WITH sha256_password BY '{{reader_password}}';

GRANT INSERT, SELECT ON audit.audit_log TO {{writer_user}};
GRANT INSERT, SELECT ON audit.change_log TO {{writer_user}};
GRANT SELECT(database, `table`, active, partition, rows) ON system.parts TO {{writer_user}};

GRANT SELECT ON audit.audit_log TO {{reader_user}};
GRANT SELECT ON audit.change_log TO {{reader_user}};
