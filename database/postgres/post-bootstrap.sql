CREATE DATABASE :"postgres_db";

\connect :postgres_db

CREATE ROLE :"readonly_role" NOLOGIN;

GRANT CONNECT ON DATABASE :"postgres_db" TO :"readonly_role";
GRANT USAGE ON SCHEMA public TO :"readonly_role";
GRANT SELECT ON ALL TABLES IN SCHEMA public TO :"readonly_role";
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO :"readonly_role";
