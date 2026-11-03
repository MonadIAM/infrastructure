locals {
  services = toset(["identity", "organization", "hr", "access-control", "notification"])
  policies = merge(
    { for namespace, binding in var.namespace_bindings : binding.policy => file("${path.module}/../../infrastructure/vault/policies/${namespace}-read.hcl") },
    { for name in local.services : "${name}-service" => file("${path.module}/../../applications/${name}/config/policy.hcl") }
  )
  roles = merge(
    { for namespace, binding in var.namespace_bindings : binding.role => {
      namespace = namespace, service_account = "vault-auth", policy = binding.policy
    } },
    { for name in local.services : "${name}-service" => {
      namespace = "monadiam-apps", service_account = "${name}-service", policy = "${name}-service"
    } }
  )
  keys = merge(
    { for name in local.services : "${name}-audit-mask-hmac" => { type = "hmac" } },
    { for name in local.services : "${name}-audit-log-sig" => { type = "ed25519" } },
    {
      identity-access-token  = { type = "ecdsa-p256" }
      identity-refresh-token = { type = "ecdsa-p256" }
      identity-id-token      = { type = "ecdsa-p256" }
      identity-totp-secret   = { type = "aes256-gcm96" }
      identity-pairwise-sub  = { type = "hmac" }
    }
  )
}

resource "vault_mount" "kv" {
  depends_on = [vault_audit.file]
  path       = var.kv_mount
  type       = "kv"
  options    = { version = "2" }
  lifecycle { prevent_destroy = true }
}

resource "vault_mount" "transit" {
  depends_on = [vault_audit.file]
  path       = "transit"
  type       = "transit"
  lifecycle { prevent_destroy = true }
}

resource "vault_auth_backend" "app" {
  depends_on = [vault_audit.file]
  path       = var.auth_mount
  type       = "kubernetes"
  lifecycle { prevent_destroy = true }
}

resource "vault_policy" "app" {
  depends_on = [vault_audit.file]
  for_each   = local.policies
  name       = each.key
  policy     = each.value
  lifecycle {
    prevent_destroy = true
    precondition {
      condition     = trimspace(each.value) != ""
      error_message = "Vault policy must not be empty."
    }
  }

}

resource "vault_kubernetes_auth_backend_config" "app" {
  backend         = vault_auth_backend.app.path
  kubernetes_host = "https://kubernetes.default.svc:443"
  lifecycle { prevent_destroy = true }
}

resource "vault_kubernetes_auth_backend_role" "app" {
  for_each                         = local.roles
  backend                          = vault_auth_backend.app.path
  role_name                        = each.key
  bound_service_account_names      = [each.value.service_account]
  bound_service_account_namespaces = [each.value.namespace]
  token_policies                   = [vault_policy.app[each.value.policy].name]
  audience                         = var.audience
  token_ttl                        = var.role_ttl
  lifecycle { prevent_destroy = true }
}

resource "vault_audit" "file" {
  count   = var.profile == "dev" ? 0 : 1
  type    = "file"
  path    = "file"
  options = { file_path = "stdout" }
}

resource "vault_transit_secret_backend_key" "app" {
  for_each         = local.keys
  backend          = vault_mount.transit.path
  name             = each.key
  type             = each.value.type
  key_size         = each.value.type == "hmac" ? 32 : 0
  deletion_allowed = false
  lifecycle {
    prevent_destroy = true
    ignore_changes  = [min_decryption_version, min_encryption_version]
  }
}
