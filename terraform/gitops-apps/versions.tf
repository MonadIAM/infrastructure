terraform {
  required_version = "= 1.17.1"
  backend "local" {}
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "3.4.0"
    }
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
provider "kubernetes" {
  config_path    = var.kubeconfig
  config_context = var.context
}
