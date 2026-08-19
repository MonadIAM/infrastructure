pid_file = "/tmp/pidfile"

vault {
    address     = "https://vault:8200"
    tls_ca_file = "/vault/tls/ca.crt"
}

auto_auth {
    method "approle" {
        mount_path = "auth/approle"

        config = {
            role_id_file_path                   = "/vault/config/role-id"
            secret_id_file_path                 = "/vault/agent-bootstrap/secret-id"
            remove_secret_id_file_after_reading = true
        }
    }

    sink "file" {
        config = {
            path = "/vault/token/agent-token"
        }
    }
}

template {
    contents    = "{{ with secret \"infra-kv/data/infra/clickhouse\" }}{{ .Data.data.clickhouse_password }}{{ end }}"
    destination = "/secrets/clickhouse/clickhouse_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"infra-kv/data/infra/clickhouse\" }}{{ .Data.data.clickhouse_writer_password }}{{ end }}"
    destination = "/secrets/clickhouse/clickhouse_writer_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"infra-kv/data/infra/clickhouse\" }}{{ .Data.data.clickhouse_reader_password }}{{ end }}"
    destination = "/secrets/clickhouse/clickhouse_reader_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"infra-kv/data/infra/minio\" }}{{ .Data.data.minio_root_password }}{{ end }}"
    destination = "/secrets/minio/minio_root_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"infra-kv/data/infra/minio\" }}{{ .Data.data.minio_archiver_secret_key }}{{ end }}"
    destination = "/secrets/minio/minio_archiver_secret_key"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"infra-kv/data/infra/grafana\" }}{{ .Data.data.admin_password }}{{ end }}"
    destination = "/secrets/grafana/admin_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"kv/data/identity-service/runtime\" }}{{ .Data.data.postgresql_password }}{{ end }}"
    destination = "/secrets/postgresql/identity/postgresql_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"kv/data/identity-service/runtime\" }}{{ .Data.data.redis_password }}{{ end }}"
    destination = "/secrets/redis/identity/redis_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"kv/data/identity-service/postgresql-replication\" }}{{ .Data.data.postgresql_replication_password }}{{ end }}"
    destination = "/secrets/postgresql/identity/postgresql_replication_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"kv/data/access-control-service/runtime\" }}{{ .Data.data.postgresql_password }}{{ end }}"
    destination = "/secrets/postgresql/access-control/postgresql_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"kv/data/access-control-service/runtime\" }}{{ .Data.data.redis_password }}{{ end }}"
    destination = "/secrets/redis/access-control/redis_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"kv/data/access-control-service/postgresql-replication\" }}{{ .Data.data.postgresql_replication_password }}{{ end }}"
    destination = "/secrets/postgresql/access-control/postgresql_replication_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"kv/data/notification-service/runtime\" }}{{ .Data.data.postgresql_password }}{{ end }}"
    destination = "/secrets/postgresql/notification/postgresql_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"kv/data/notification-service/runtime\" }}{{ .Data.data.redis_password }}{{ end }}"
    destination = "/secrets/redis/notification/redis_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"kv/data/notification-service/postgresql-replication\" }}{{ .Data.data.postgresql_replication_password }}{{ end }}"
    destination = "/secrets/postgresql/notification/postgresql_replication_password"
    perms       = "0440"
}
