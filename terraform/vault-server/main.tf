locals {
  issuers = [for document in split("\n---\n", trimspace(file("${path.module}/../../infrastructure/cert-manager/base/issuers.yaml"))) : yamldecode(document)]
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
  manifest   = yamldecode(file("${path.module}/../../infrastructure/vault/base/certificate.yaml"))
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
    file("${path.module}/../../infrastructure/vault/values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/infrastructure/vault/values.yaml")
  ]
  depends_on = [kubernetes_manifest.vault_tls]
  wait       = true
  timeout    = 300
  lifecycle { prevent_destroy = true }
}
