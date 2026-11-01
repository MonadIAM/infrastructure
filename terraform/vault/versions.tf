terraform {
  required_version = "= 1.17.1"
  required_providers {
    vault = {
      source  = "hashicorp/vault"
      version = "= 5.13.0"
    }
  }
  backend "local" {}
}

provider "vault" {}

variable "kv_mount" { type = string }
variable "auth_mount" { type = string }
variable "audience" { type = string }
variable "role_ttl" {
  type = number
  validation {
    condition     = var.role_ttl > 0
    error_message = "role_ttl must be positive."
  }
}
variable "namespace_bindings" {
  type = map(object({ policy = string, role = string }))
}
