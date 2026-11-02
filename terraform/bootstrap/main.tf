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

resource "kubernetes_namespace_v1" "cert_manager" {
  metadata {
    name = "cert-manager"
    labels = {
      "app.kubernetes.io/name"       = "cert-manager"
      "app.kubernetes.io/component"  = "pki"
      "app.kubernetes.io/part-of"    = "monadiam"
      "app.kubernetes.io/managed-by" = "terraform"
      "istio-injection"              = "disabled"
    }
  }
  lifecycle { prevent_destroy = true }
}

resource "kubernetes_namespace_v1" "vault" {
  metadata {
    name = "vault"
    labels = {
      "app.kubernetes.io/name"       = "vault"
      "app.kubernetes.io/component"  = "secrets"
      "app.kubernetes.io/part-of"    = "monadiam"
      "app.kubernetes.io/managed-by" = "terraform"
      "istio-injection"              = "disabled"
      "istio.io/dataplane-mode"      = "ambient"
    }
  }
  lifecycle { prevent_destroy = true }
}

resource "kubernetes_storage_class_v1" "vault" {
  metadata {
    name = "local-path-retain"
    labels = {
      "app.kubernetes.io/name"       = "local-path-retain"
      "app.kubernetes.io/component"  = "storage"
      "app.kubernetes.io/part-of"    = "monadiam"
      "app.kubernetes.io/managed-by" = "terraform"
    }
  }
  storage_provisioner = "rancher.io/local-path"
  reclaim_policy      = "Retain"
  volume_binding_mode = "WaitForFirstConsumer"
  lifecycle { prevent_destroy = true }
}

resource "helm_release" "cert_manager" {
  name      = "cert-manager"
  namespace = kubernetes_namespace_v1.cert_manager.metadata[0].name
  chart     = var.cert_manager_chart
  version   = var.cert_manager_version
  values = [
    file("${path.module}/../../infrastructure/security/cert-manager/helm/values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/cert-manager/values.yaml")
  ]
  wait    = true
  timeout = 300
  lifecycle { prevent_destroy = true }
}
