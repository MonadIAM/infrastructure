[databases]
${IDENTITY_POSTGRES_DB} = host=postgres-haproxy port=${IDENTITY_POSTGRES_TARGET_PORT} dbname=${IDENTITY_POSTGRES_DB} auth_user=${IDENTITY_POSTGRES_USER}
${ACCESS_CONTROL_POSTGRES_DB} = host=postgres-haproxy port=${ACCESS_CONTROL_POSTGRES_TARGET_PORT} dbname=${ACCESS_CONTROL_POSTGRES_DB} auth_user=${ACCESS_CONTROL_POSTGRES_USER}
${NOTIFICATION_POSTGRES_DB} = host=postgres-haproxy port=${NOTIFICATION_POSTGRES_TARGET_PORT} dbname=${NOTIFICATION_POSTGRES_DB} auth_user=${NOTIFICATION_POSTGRES_USER}

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = ${PGBOUNCER_LISTEN_PORT}
auth_type = md5
auth_file = /tmp/userlist.txt
auth_query = SELECT username, password FROM pgbouncer.get_auth($1)
pool_mode = transaction
server_reset_query = DISCARD ALL
ignore_startup_parameters = extra_float_digits
max_client_conn = ${PGBOUNCER_MAX_CLIENT_CONN}
default_pool_size = ${PGBOUNCER_DEFAULT_POOL_SIZE}
reserve_pool_size = ${PGBOUNCER_RESERVE_POOL_SIZE}
pidfile = /tmp/pgbouncer.pid
user = pgbouncer
