terraform {
  required_version = "= 1.17.1"
  backend "local" {}
  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "3.3.0"
    }
  }
}

variable "kubeconfig" {
  type = string
}

variable "context" {
  type = string
}

variable "profile" {
  type = string
}

provider "helm" {
  kubernetes = {
    config_path    = var.kubeconfig
    config_context = var.context
  }
}
