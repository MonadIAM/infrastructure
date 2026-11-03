resource "kubernetes_namespace_v1" "argocd" {
  metadata {
    name = "argocd"
    labels = {
      "app.kubernetes.io/part-of"    = "monadiam"
      "app.kubernetes.io/managed-by" = "terraform"
      "istio-injection"              = "disabled"
    }
  }
  lifecycle { prevent_destroy = true }
}

resource "helm_release" "argocd" {
  name       = "argocd"
  namespace  = kubernetes_namespace_v1.argocd.metadata[0].name
  repository = "https://argoproj.github.io/argo-helm"
  chart      = "argo-cd"
  version    = "10.11.6"
  values = [
    file("${path.module}/../../infrastructure/argocd/values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/infrastructure/argocd/values.yaml")
  ]
  wait    = true
  timeout = 600
  lifecycle { prevent_destroy = true }
}

resource "helm_release" "image_updater" {
  name       = "argocd-image-updater"
  namespace  = kubernetes_namespace_v1.argocd.metadata[0].name
  repository = "https://argoproj.github.io/argo-helm"
  chart      = "argocd-image-updater"
  version    = "1.4.0"
  values = [
    file("${path.module}/../../infrastructure/image-updater/values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/infrastructure/image-updater/values.yaml")
  ]
  depends_on = [helm_release.argocd]
  wait       = true
  timeout    = 300
  lifecycle { prevent_destroy = true }
}
