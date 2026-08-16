pid_file = "/tmp/pidfile"

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
