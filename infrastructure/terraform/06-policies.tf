resource "vault_policy" "service_secrets" {
    for_each = local.services
    name     = "${each.key}-secrets"

    policy = join("\n", compact([
        <<-EOT
        path "${local.kv_mount}/data/${each.value.kv_runtime_path}" {
          capabilities = ["read"]
        }

        path "${local.database_mount}/creds/${each.key}-app" {
          capabilities = ["read"]
        }
        EOT
        ,
        each.value.kv_oauth_path != null ? <<-EOT
        path "${local.kv_mount}/data/${each.value.kv_oauth_path}" {
          capabilities = ["read"]
        }
        EOT
        : "",
        each.value.has_aws ? <<-EOT
        path "${local.aws_mount}/creds/${each.key}-ses" {
          capabilities = ["read"]
        }
        EOT
        : "",
    ]))
}

resource "vault_policy" "service_developer_readonly" {
    for_each = local.services
    name     = "${each.key}-developer-readonly"

    policy = <<-EOT
        path "${local.database_mount}/creds/${each.key}-developer-readonly" {
          capabilities = ["read"]
        }
    EOT
}

resource "vault_policy" "identity_service_transit" {
    name = "identity-service"

    policy = <<-EOT
        path "${local.transit_mount}/keys/${var.identity_transit_access_key}" {
          capabilities = ["read"]
        }

        path "${local.transit_mount}/keys/${var.identity_transit_refresh_key}" {
          capabilities = ["read"]
        }

        path "${local.transit_mount}/keys/${var.identity_transit_id_key}" {
          capabilities = ["read"]
        }

        path "${local.transit_mount}/keys/${var.identity_transit_totp_key}" {
          capabilities = ["read"]
        }

        path "${local.transit_mount}/sign/${var.identity_transit_access_key}" {
          capabilities = ["update"]
        }

        path "${local.transit_mount}/sign/${var.identity_transit_refresh_key}" {
          capabilities = ["update"]
        }

        path "${local.transit_mount}/sign/${var.identity_transit_id_key}" {
          capabilities = ["update"]
        }

        path "${local.transit_mount}/encrypt/${var.identity_transit_totp_key}" {
          capabilities = ["update"]
        }

        path "${local.transit_mount}/decrypt/${var.identity_transit_totp_key}" {
          capabilities = ["update"]
        }

        path "${local.transit_mount}/rewrap/${var.identity_transit_totp_key}" {
          capabilities = ["update"]
        }

        path "${local.transit_mount}/hmac/${var.identity_transit_pairwise_key}" {
          capabilities = ["update"]
        }
    EOT
}

resource "vault_policy" "identity_service_key_trim" {
    name = "identity-service-key-trim"

    policy = <<-EOT
        path "${local.transit_mount}/keys/${var.identity_transit_access_key}" {
          capabilities = ["read"]
        }

        path "${local.transit_mount}/keys/${var.identity_transit_refresh_key}" {
          capabilities = ["read"]
        }

        path "${local.transit_mount}/keys/${var.identity_transit_id_key}" {
          capabilities = ["read"]
        }

        path "${local.transit_mount}/keys/${var.identity_transit_access_key}/config" {
          capabilities = ["update"]
        }

        path "${local.transit_mount}/keys/${var.identity_transit_refresh_key}/config" {
          capabilities = ["update"]
        }

        path "${local.transit_mount}/keys/${var.identity_transit_id_key}/config" {
          capabilities = ["update"]
        }

        path "${local.transit_mount}/keys/${var.identity_transit_access_key}/trim" {
          capabilities = ["update"]
        }

        path "${local.transit_mount}/keys/${var.identity_transit_refresh_key}/trim" {
          capabilities = ["update"]
        }

        path "${local.transit_mount}/keys/${var.identity_transit_id_key}/trim" {
          capabilities = ["update"]
        }

        path "auth/token/renew-self" {
          capabilities = ["update"]
        }
    EOT
}

resource "vault_policy" "infra_secrets" {
    name = "infra-secrets"

    policy = <<-EOT
        path "${local.infra_kv_mount}/data/infra/clickhouse" {
          capabilities = ["read"]
        }

        path "${local.infra_kv_mount}/data/infra/minio" {
          capabilities = ["read"]
        }

        path "${local.infra_kv_mount}/data/infra/grafana" {
          capabilities = ["read"]
        }

        path "${local.kv_mount}/data/identity-service/runtime" {
          capabilities = ["read"]
        }

        path "${local.kv_mount}/data/identity-service/postgresql-replication" {
          capabilities = ["read"]
        }

        path "${local.kv_mount}/data/access-control-service/runtime" {
          capabilities = ["read"]
        }

        path "${local.kv_mount}/data/access-control-service/postgresql-replication" {
          capabilities = ["read"]
        }

        path "${local.kv_mount}/data/notification-service/runtime" {
          capabilities = ["read"]
        }

        path "${local.kv_mount}/data/notification-service/postgresql-replication" {
          capabilities = ["read"]
        }
    EOT
}
