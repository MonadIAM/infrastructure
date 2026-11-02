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

locals {
  issuers = [for document in split("\n---\n", trimspace(file("${path.module}/../../infrastructure/security/cert-manager/resources/issuers.yaml"))) : yamldecode(document)]
}

resource "kubernetes_manifest" "selfsigned" {
  manifest = local.issuers[0]
  wait {
    condition {
      type   = "Ready"
      status = "True"
    }
  }
  timeouts {
    create = "3m"
    update = "3m"
  }
  lifecycle { prevent_destroy = true }
}
resource "kubernetes_manifest" "root_ca" {
  manifest   = local.issuers[1]
  depends_on = [kubernetes_manifest.selfsigned]
  wait {
    condition {
      type   = "Ready"
      status = "True"
    }
  }
  timeouts {
    create = "3m"
    update = "3m"
  }
  lifecycle { prevent_destroy = true }
}
resource "kubernetes_manifest" "ca" {
  manifest   = local.issuers[2]
  depends_on = [kubernetes_manifest.root_ca]
  wait {
    condition {
      type   = "Ready"
      status = "True"
    }
  }
  timeouts {
    create = "3m"
    update = "3m"
  }
  lifecycle { prevent_destroy = true }
}
resource "kubernetes_manifest" "vault_tls" {
  manifest   = yamldecode(file("${path.module}/../../platform/vault/resources/certificate.yaml"))
  depends_on = [kubernetes_manifest.ca]
  wait {
    condition {
      type   = "Ready"
      status = "True"
    }
  }
  timeouts {
    create = "3m"
    update = "3m"
  }
  lifecycle { prevent_destroy = true }
}
resource "helm_release" "vault" {
  name       = "vault"
  namespace  = "vault"
  repository = var.hashicorp_repository
  chart      = "vault"
  version    = var.vault_chart_version
  values = [
    file("${path.module}/../../platform/vault/helm/values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/vault/values.yaml")
  ]
  depends_on = [kubernetes_manifest.vault_tls]
  wait       = true
  timeout    = 300
  lifecycle { prevent_destroy = true }
}
