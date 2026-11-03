variable "profile" {
  type = string
}

variable "cert_manager_chart" {
  type = string
  validation {
    condition     = trimspace(var.cert_manager_chart) != ""
    error_message = "cert_manager_chart is required."
  }
}

variable "cert_manager_version" {
  type = string
  validation {
    condition     = trimspace(var.cert_manager_version) != ""
    error_message = "cert_manager_version is required."
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
