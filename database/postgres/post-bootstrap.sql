CREATE DATABASE :"postgres_db";

\connect :postgres_db

CREATE ROLE :"readonly_role" NOLOGIN;

GRANT CONNECT ON DATABASE :"postgres_db" TO :"readonly_role";
GRANT USAGE ON SCHEMA public TO :"readonly_role";
GRANT SELECT ON ALL TABLES IN SCHEMA public TO :"readonly_role";
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO :"readonly_role";

CREATE SCHEMA IF NOT EXISTS pgbouncer;

CREATE OR REPLACE FUNCTION pgbouncer.get_auth(p_usename TEXT)
RETURNS TABLE(username TEXT, password TEXT)
LANGUAGE SQL
SECURITY DEFINER
SET search_path = pg_catalog
AS 'SELECT rolname::TEXT, rolpassword::TEXT FROM pg_authid WHERE rolname = p_usename;';

REVOKE ALL ON SCHEMA pgbouncer FROM PUBLIC;
REVOKE ALL ON FUNCTION pgbouncer.get_auth(TEXT) FROM PUBLIC;
GRANT USAGE ON SCHEMA pgbouncer TO :"postgres_user";
GRANT EXECUTE ON FUNCTION pgbouncer.get_auth(TEXT) TO :"postgres_user";
