resource "vault_transit_secret_backend_key" "access" {
    backend            = vault_mount.transit.path
    name               = var.identity_transit_access_key
    type               = "ecdsa-p256"
    auto_rotate_period = var.identity_transit_access_rotation_period
    deletion_allowed   = true
}

resource "vault_transit_secret_backend_key" "refresh" {
    backend            = vault_mount.transit.path
    name               = var.identity_transit_refresh_key
    type               = "ecdsa-p256"
    auto_rotate_period = var.identity_transit_refresh_rotation_period
    deletion_allowed   = true
}

resource "vault_transit_secret_backend_key" "id_token" {
    backend            = vault_mount.transit.path
    name               = var.identity_transit_id_key
    type               = "ecdsa-p256"
    auto_rotate_period = var.identity_transit_id_rotation_period
    deletion_allowed   = true
}

resource "vault_transit_secret_backend_key" "totp" {
    backend            = vault_mount.transit.path
    name               = var.identity_transit_totp_key
    type               = "aes256-gcm96"
    auto_rotate_period = var.identity_transit_totp_rotation_period
}

resource "vault_transit_secret_backend_key" "pairwise" {
    backend  = vault_mount.transit.path
    name     = var.identity_transit_pairwise_key
    type     = "hmac"
    key_size = 32
}
