scope: ${PATRONI_SCOPE}
namespace: /service/

restapi:
  listen: 0.0.0.0:${PATRONI_RESTAPI_PORT}

consul:
  host: consul:${CONSUL_HTTP_PORT}
  register_service: true

bootstrap:
  dcs:
    ttl: 30
    loop_wait: 10
    retry_timeout: 10
    maximum_lag_on_failover: 1048576
    postgresql:
      use_pg_rewind: true
      use_slots: true
      parameters:
        ssl: "on"
        ssl_cert_file: ${POSTGRES_SSL_CERT_FILE}
        ssl_key_file: ${POSTGRES_SSL_KEY_FILE}
        ssl_ca_file: ${POSTGRES_SSL_CA_FILE}
        wal_level: logical
        max_replication_slots: 10
        max_wal_senders: 10
        hot_standby: "on"
      slots:
        ${POSTGRES_OUTBOX_SLOT}:
          type: logical
          database: ${POSTGRES_DB}
          plugin: pgoutput

  initdb:
    - encoding: UTF8
    - data-checksums

  pg_hba:
    - hostssl replication all 0.0.0.0/0 md5
    - hostssl all all 0.0.0.0/0 md5

  post_bootstrap: /docker-entrypoint-initdb.d/post-bootstrap.sh

postgresql:
  listen: 0.0.0.0:5432
  data_dir: /var/lib/postgresql/data/pgdata
  pgpass: /tmp/pgpass
  pg_hba:
    - hostssl replication all 0.0.0.0/0 md5
    - hostssl all all 0.0.0.0/0 md5
  parameters:
    ssl: "on"
    ssl_ca_file: ${POSTGRES_SSL_CA_FILE}
    ssl_cert_file: ${POSTGRES_SSL_CERT_FILE}
    ssl_key_file: ${POSTGRES_SSL_KEY_FILE}
    unix_socket_directories: /var/run/postgresql

  create_replica_methods:
    - basebackup

  basebackup:
    checkpoint: 'fast'

  remove_data_directory_on_rewind_failure: true

tags:
  noloadbalance: false
  nofailover: false
  clonefrom: false
  nosync: false
