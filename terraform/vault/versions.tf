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
