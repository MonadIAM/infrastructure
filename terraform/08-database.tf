resource "vault_database_secret_backend_connection" "service" {
    for_each = local.services

    backend       = vault_mount.database.path
    name          = "${each.key}-postgres"
    allowed_roles = ["${each.key}-app", "${each.key}-developer-readonly"]

    postgresql {
        connection_url = "postgresql://{{username}}:{{password}}@${each.value.postgres_host}:${each.value.postgres_port}/${each.value.postgres_db}?sslmode=disable"
        username       = each.value.postgres_admin_user
        password       = random_password.postgres_admin[each.key].result
    }

    verify_connection = false
}

resource "vault_database_secret_backend_role" "app" {
    for_each = local.services

    backend = vault_mount.database.path
    name    = "${each.key}-app"
    db_name = vault_database_secret_backend_connection.service[each.key].name

    creation_statements = [
        "CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}' IN ROLE \"${each.value.postgres_admin_user}\";",
    ]
    revocation_statements = [
        "DROP ROLE IF EXISTS \"{{name}}\";",
    ]

    default_ttl = 3600
    max_ttl     = 86400
}

resource "vault_database_secret_backend_role" "developer_readonly" {
    for_each = local.services

    backend = vault_mount.database.path
    name    = "${each.key}-developer-readonly"
    db_name = vault_database_secret_backend_connection.service[each.key].name

    creation_statements = [
        "CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}' IN ROLE \"${replace(each.key, "-", "_")}_readonly\";",
    ]
    revocation_statements = [
        "DROP ROLE IF EXISTS \"{{name}}\";",
    ]

    default_ttl = 1800
    max_ttl     = 14400
}
