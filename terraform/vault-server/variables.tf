variable "profile" {
  type = string
}

variable "vault_chart_version" {
  type = string
  validation {
    condition     = trimspace(var.vault_chart_version) != ""
    error_message = "vault_chart_version is required."
  }
}

variable "hashicorp_repository" {
  type = string
  validation {
    condition     = trimspace(var.hashicorp_repository) != ""
    error_message = "hashicorp_repository is required."
  }
}

variable "kubeconfig" {
  type = string
  validation {
    condition     = trimspace(var.kubeconfig) != ""
    error_message = "kubeconfig is required."
  }
}

variable "context" {
  type = string
  validation {
    condition     = trimspace(var.context) != ""
    error_message = "context is required."
  }
}
