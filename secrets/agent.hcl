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
    contents    = "{{ with secret \"infra-kv/data/infra/kafka\" }}{{ .Data.data.kafka_broker_sasl_password }}{{ end }}"
    destination = "/secrets/kafka/kafka_broker_sasl_password"
    perms       = "0440"
    group       = "10000"
}

template {
    contents    = "{{ with secret \"infra-kv/data/infra/kafka\" }}{{ .Data.data.kafka_admin_sasl_password }}{{ end }}"
    destination = "/secrets/kafka/kafka_admin_sasl_password"
    perms       = "0440"
    group       = "10000"
}

template {
    contents    = "{{ with secret \"infra-kv/data/infra/kafka\" }}{{ .Data.data.kafka_connect_sasl_password }}{{ end }}"
    destination = "/secrets/kafka/kafka_connect_sasl_password"
    perms       = "0440"
    group       = "10000"
}

template {
    contents    = "{{ with secret \"infra-kv/data/infra/kafka\" }}{{ .Data.data.kafka_ui_sasl_password }}{{ end }}"
    destination = "/secrets/kafka/kafka_ui_sasl_password"
    perms       = "0440"
    group       = "10000"
}

template {
    contents    = "{{ with secret \"infra-kv/data/infra/kafka\" }}{{ .Data.data.kafka_ui_admin_password }}{{ end }}"
    destination = "/secrets/kafka/kafka_ui_admin_password"
    perms       = "0440"
    group       = "10000"
}

template {
    contents    = "{{ with secret \"infra-kv/data/infra/kafka\" }}{{ .Data.data.kafka_clickhouse_sasl_password }}{{ end }}"
    destination = "/secrets/kafka/kafka_clickhouse_sasl_password"
    perms       = "0440"
    group       = "10000"
}

template {
    contents    = "{{ with secret \"infra-kv/data/infra/kafka\" }}{{ .Data.data.kafka_clickhouse_sasl_password }}{{ end }}"
    destination = "/secrets/clickhouse/kafka_sasl_password"
    perms       = "0440"
}

template {
    contents    = <<EOH
{{ with secret "infra-kv/data/infra/kafka" }}
auth:
  type: LOGIN_FORM
spring:
  security:
    user:
      name: dev_kafka_admin
      password: {{ .Data.data.kafka_ui_admin_password }}
kafka:
  clusters:
    - name: local
      bootstrapServers: kafka:29092
      properties:
        security.protocol: SASL_SSL
        sasl.mechanism: SCRAM-SHA-512
        sasl.jaas.config: org.apache.kafka.common.security.scram.ScramLoginModule required username="kafka-ui" password="{{ .Data.data.kafka_ui_sasl_password }}";
        ssl.keystore.type: PEM
        ssl.keystore.location: /tls/client.pem
        ssl.truststore.type: PEM
        ssl.truststore.location: /tls/ca.crt
        ssl.endpoint.identification.algorithm: https
{{ end }}
EOH
    destination = "/secrets/kafka/kafka-ui.yml"
    perms       = "0440"
    group       = "10000"
}

template {
    contents    = "{{ with secret \"kv/data/identity-service/runtime\" }}{{ .Data.data.postgresql_password }}{{ end }}"
    destination = "/secrets/postgresql/identity/postgresql_password"
    perms       = "0440"
    group       = "10000"
}

template {
    contents    = "{{ with secret \"kv/data/identity-service/runtime\" }}{{ .Data.data.redis_password }}{{ end }}"
    destination = "/secrets/redis/identity/redis_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"kv/data/identity-service/runtime\" }}{{ .Data.data.kafka_sasl_password }}{{ end }}"
    destination = "/secrets/kafka/identity/kafka_sasl_password"
    perms       = "0440"
    group       = "10000"
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
    group       = "10000"
}

template {
    contents    = "{{ with secret \"kv/data/access-control-service/runtime\" }}{{ .Data.data.redis_password }}{{ end }}"
    destination = "/secrets/redis/access-control/redis_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"kv/data/access-control-service/runtime\" }}{{ .Data.data.kafka_sasl_password }}{{ end }}"
    destination = "/secrets/kafka/access-control/kafka_sasl_password"
    perms       = "0440"
    group       = "10000"
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
    group       = "10000"
}

template {
    contents    = "{{ with secret \"kv/data/notification-service/runtime\" }}{{ .Data.data.redis_password }}{{ end }}"
    destination = "/secrets/redis/notification/redis_password"
    perms       = "0440"
}

template {
    contents    = "{{ with secret \"kv/data/notification-service/runtime\" }}{{ .Data.data.kafka_sasl_password }}{{ end }}"
    destination = "/secrets/kafka/notification/kafka_sasl_password"
    perms       = "0440"
    group       = "10000"
}

template {
    contents    = "{{ with secret \"kv/data/notification-service/postgresql-replication\" }}{{ .Data.data.postgresql_replication_password }}{{ end }}"
    destination = "/secrets/postgresql/notification/postgresql_replication_password"
    perms       = "0440"
}
