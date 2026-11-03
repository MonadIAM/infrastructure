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
