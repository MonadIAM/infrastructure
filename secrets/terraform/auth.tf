resource "vault_auth_backend" "approle" {
    type = "approle"
    path = "approle"
}

resource "vault_approle_auth_backend_role" "service" {
    for_each  = local.services
    backend   = vault_auth_backend.approle.path
    role_name = each.key

    token_policies = concat(
        ["${each.key}-secrets"],
        each.value.has_transit ? [each.key] : [],
    )

    token_ttl     = 900
    token_max_ttl = 3600

    secret_id_ttl      = 600
    secret_id_num_uses = 1
}

resource "vault_approle_auth_backend_role" "infra" {
    backend   = vault_auth_backend.approle.path
    role_name = "infra"

    token_policies = [vault_policy.infra_secrets.name]
    token_ttl      = 900
    token_max_ttl  = 3600

    secret_id_ttl      = 600
    secret_id_num_uses = 1
}

resource "vault_approle_auth_backend_role" "identity_key_trim" {
    backend   = vault_auth_backend.approle.path
    role_name = "identity-service-key-trim"

    token_policies = ["identity-service-key-trim"]
    token_period   = 172800

    secret_id_ttl      = 600
    secret_id_num_uses = 1
}

resource "vault_auth_backend" "kubernetes" {
    count = var.enable_kubernetes_auth ? 1 : 0
    type  = "kubernetes"
    path  = "kubernetes"
}

resource "vault_kubernetes_auth_backend_config" "this" {
    count              = var.enable_kubernetes_auth ? 1 : 0
    backend            = vault_auth_backend.kubernetes[0].path
    kubernetes_host    = var.kubernetes_host
    kubernetes_ca_cert = var.kubernetes_ca_cert
}

resource "vault_kubernetes_auth_backend_role" "service" {
    for_each  = var.enable_kubernetes_auth ? local.services : {}
    backend   = vault_auth_backend.kubernetes[0].path
    role_name = each.key

    bound_service_account_names      = [each.value.kubernetes_service_account]
    bound_service_account_namespaces = [each.value.kubernetes_namespace]

    token_policies = concat(
        ["${each.key}-secrets"],
        each.value.has_transit ? [each.key] : [],
    )
    token_ttl     = 900
    token_max_ttl = 3600
}

resource "vault_kubernetes_auth_backend_role" "identity_key_trim" {
    count     = var.enable_kubernetes_auth ? 1 : 0
    backend   = vault_auth_backend.kubernetes[0].path
    role_name = "identity-service-key-trim"

    bound_service_account_names      = ["identity-service-key-trim"]
    bound_service_account_namespaces = [local.services["identity-service"].kubernetes_namespace]
    token_policies                   = ["identity-service-key-trim"]
    token_period                     = 172800
}

resource "vault_policy" "provisioner" {
    for_each = local.services
    name     = "${each.key}-provisioner"

    policy = <<-EOT
        path "auth/approle/role/${each.key}/role-id" {
          capabilities = ["read"]
        }

        path "auth/approle/role/${each.key}/secret-id" {
          capabilities = ["create", "update"]
        }
        %{if each.value.has_key_trim~}
        path "auth/approle/role/${each.key}-key-trim/role-id" {
          capabilities = ["read"]
        }

        path "auth/approle/role/${each.key}-key-trim/secret-id" {
          capabilities = ["create", "update"]
        }
        %{endif~}
    EOT
}

resource "vault_token" "provisioner" {
    for_each          = local.services
    policies          = [vault_policy.provisioner[each.key].name]
    no_default_policy = true
    renewable         = true
    ttl               = "720h"
}


resource "vault_policy" "infra_agent_bootstrap" {
    name = "infra-agent-bootstrap"

    policy = <<-EOT
        path "auth/approle/role/infra/role-id" {
          capabilities = ["read"]
        }

        path "auth/approle/role/infra/secret-id" {
          capabilities = ["create", "update"]
        }

        path "auth/token/renew-self" {
          capabilities = ["update"]
        }

        path "auth/token/lookup-self" {
          capabilities = ["read"]
        }
    EOT
}

resource "vault_token" "infra_agent_bootstrap" {
    policies          = [vault_policy.infra_agent_bootstrap.name]
    no_default_policy = true
    renewable         = true
    ttl               = "720h"
}
