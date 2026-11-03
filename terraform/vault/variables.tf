variable "profile" { type = string }

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
