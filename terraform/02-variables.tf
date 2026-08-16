variable "vault_address" {
    type    = string
    default = "http://127.0.0.1:8200"
}

variable "vault_bootstrap_token" {
    type      = string
    sensitive = true
}

variable "environment" {
    type    = string
    default = "dev"

    validation {
        condition     = contains(["dev", "stand"], var.environment)
        error_message = "environment must be one of: dev, stand."
    }
}

variable "enable_kubernetes_auth" {
    type    = bool
    default = false
}

variable "kubernetes_host" {
    type    = string
    default = ""
}

variable "kubernetes_ca_cert" {
    type      = string
    default   = ""
    sensitive = true
}

variable "identity_oauth_google_client_secret" {
    type      = string
    default   = ""
    sensitive = true
}

variable "identity_oauth_github_client_secret" {
    type      = string
    default   = ""
    sensitive = true
}

variable "identity_transit_access_key" {
    type    = string
    default = "identity-access-token"
}

variable "identity_transit_access_rotation_period" {
    type    = number
    default = 86400
}

variable "identity_transit_refresh_key" {
    type    = string
    default = "identity-refresh-token"
}

variable "identity_transit_refresh_rotation_period" {
    type    = number
    default = 604800
}

variable "identity_transit_id_key" {
    type    = string
    default = "identity-id-token"
}

variable "identity_transit_id_rotation_period" {
    type    = number
    default = 86400
}

variable "identity_transit_totp_key" {
    type    = string
    default = "identity-totp-secret"
}

variable "identity_transit_totp_rotation_period" {
    type    = number
    default = 2592000
}

variable "identity_transit_pairwise_key" {
    type    = string
    default = "identity-pairwise-sub"
}
