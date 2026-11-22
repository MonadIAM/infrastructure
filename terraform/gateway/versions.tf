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
