resource "random_password" "postgres_admin" {
    for_each = local.services
    length   = 32
    special  = false
}

resource "random_password" "postgres_replication" {
    for_each = local.services
    length   = 32
    special  = false
}

resource "random_password" "redis" {
    for_each = local.services
    length   = 32
    special  = false
}

resource "vault_kv_secret_v2" "service_runtime" {
    for_each = local.services
    mount    = vault_mount.kv.path
    name     = each.value.kv_runtime_path

    data_json = jsonencode({
        postgresql_password = random_password.postgres_admin[each.key].result
        redis_password      = random_password.redis[each.key].result
    })
}

resource "vault_kv_secret_v2" "service_replication" {
    for_each = local.services
    mount    = vault_mount.kv.path
    name     = each.value.kv_replication_path

    data_json = jsonencode({
        postgresql_replication_password = random_password.postgres_replication[each.key].result
    })
}

resource "vault_kv_secret_v2" "identity_oauth" {
    mount = vault_mount.kv.path
    name  = local.services["identity-service"].kv_oauth_path

    data_json = jsonencode({
        google_client_secret = var.identity_oauth_google_client_secret
        github_client_secret = var.identity_oauth_github_client_secret
    })
}

resource "random_password" "clickhouse_default" {
    length  = 32
    special = false
}

resource "random_password" "clickhouse_writer" {
    length  = 32
    special = false
}

resource "random_password" "clickhouse_reader" {
    length  = 32
    special = false
}

resource "random_password" "minio_root" {
    length  = 32
    special = false
}

resource "random_password" "minio_archiver_secret_key" {
    length  = 32
    special = false
}

resource "random_password" "grafana_admin" {
    length  = 24
    special = false
}

resource "vault_kv_secret_v2" "infra_clickhouse" {
    mount = vault_mount.infra_kv.path
    name  = "infra/clickhouse"

    data_json = jsonencode({
        clickhouse_password        = random_password.clickhouse_default.result
        clickhouse_writer_password = random_password.clickhouse_writer.result
        clickhouse_reader_password = random_password.clickhouse_reader.result
    })
}

resource "vault_kv_secret_v2" "infra_minio" {
    mount = vault_mount.infra_kv.path
    name  = "infra/minio"

    data_json = jsonencode({
        minio_root_password       = random_password.minio_root.result
        minio_archiver_secret_key = random_password.minio_archiver_secret_key.result
    })
}

resource "vault_kv_secret_v2" "infra_grafana" {
    mount = vault_mount.infra_kv.path
    name  = "infra/grafana"

    data_json = jsonencode({
        admin_password = random_password.grafana_admin.result
    })
}
