locals {
  services = toset(["identity", "template", "organization", "hr", "access-control", "notification"])
  policies = merge(
    { for name in local.services : "${name}-service" => file("${var.policies_dir}/${name}-service.hcl") },
    { for name in local.services : "${name}-service-transit" => file("${var.policies_dir}/${name}-service-transit.hcl") }
  )
  keys = merge(
    { for name in local.services : "${name}-audit-mask-hmac" => { type = "hmac", signing = false } },
    { for name in local.services : "${name}-audit-log-sig" => { type = "ed25519", signing = true } },
    {
      identity-access-token   = { type = "ecdsa-p256", signing = true }
      identity-refresh-token  = { type = "ecdsa-p256", signing = true }
      identity-id-token       = { type = "ecdsa-p256", signing = true }
      identity-id-token-rs256 = { type = "rsa-2048", signing = true }
      identity-totp-secret    = { type = "aes256-gcm96", signing = false }
      identity-pairwise-sub   = { type = "hmac", signing = false }
    }
  )
}

resource "vault_mount" "kv" {
  path    = "kv"
  type    = "kv"
  options = { version = "2" }
  lifecycle { prevent_destroy = true }
}

resource "vault_mount" "transit" {
  path = "transit"
  type = "transit"
  lifecycle { prevent_destroy = true }
}

resource "vault_auth_backend" "app" {
  path = "approle"
  type = "approle"
  lifecycle { prevent_destroy = true }
}

resource "vault_policy" "app" {
  for_each = local.policies
  name     = each.key
  policy   = each.value
  lifecycle {
    prevent_destroy = true
    precondition {
      condition     = trimspace(each.value) != ""
      error_message = "Vault policy must not be empty."
    }
  }

}

resource "vault_approle_auth_backend_role" "app" {
  for_each           = local.services
  backend            = vault_auth_backend.app.path
  role_name          = "${each.key}-service"
  token_policies     = [vault_policy.app["${each.key}-service"].name, vault_policy.app["${each.key}-service-transit"].name]
  token_ttl          = 3600
  token_max_ttl      = 86400
  secret_id_ttl      = 0
  secret_id_num_uses = 0
  lifecycle { prevent_destroy = true }
}

resource "vault_transit_secret_backend_key" "app" {
  for_each         = local.keys
  backend          = vault_mount.transit.path
  name             = each.key
  type             = each.value.type
  key_size         = each.value.type == "hmac" ? 32 : 0
  deletion_allowed = each.value.signing
  lifecycle {
    prevent_destroy = true
    ignore_changes  = [min_decryption_version, min_encryption_version]
  }
}

