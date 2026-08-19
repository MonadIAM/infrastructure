terraform {
    required_version = ">= 1.9.0"

    required_providers {
        vault = {
            source  = "hashicorp/vault"
            version = "~> 4.4"
        }
        random = {
            source  = "hashicorp/random"
            version = "~> 3.6"
        }
    }
}
