resource "vault_mount" "kv" {
    path = local.kv_mount
    type = "kv-v2"
}

resource "vault_mount" "infra_kv" {
    path = local.infra_kv_mount
    type = "kv-v2"
}

resource "vault_mount" "database" {
    path = local.database_mount
    type = "database"
}

resource "vault_mount" "transit" {
    path = local.transit_mount
    type = "transit"
}
