resource "helm_release" "base" {
  name       = "istio-base"
  namespace  = "istio-system"
  repository = "https://istio-release.storage.googleapis.com/charts"
  chart      = "base"
  version    = "1.30.3"
  values = [
    file("${path.module}/../../infrastructure/istio/base-values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/infrastructure/istio/values.yaml")
  ]
  wait    = true
  timeout = 300
  lifecycle { prevent_destroy = true }
}

resource "helm_release" "istiod" {
  name       = "istiod"
  namespace  = "istio-system"
  repository = "https://istio-release.storage.googleapis.com/charts"
  chart      = "istiod"
  version    = "1.30.3"
  values = [
    file("${path.module}/../../infrastructure/istio/istiod-values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/infrastructure/istio/values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/infrastructure/istio/istiod-values.yaml")
  ]
  depends_on = [helm_release.base]
  wait       = true
  timeout    = 300
  lifecycle { prevent_destroy = true }
}

resource "helm_release" "cni" {
  name       = "istio-cni"
  namespace  = "istio-system"
  repository = "https://istio-release.storage.googleapis.com/charts"
  chart      = "cni"
  version    = "1.30.3"
  values = [
    file("${path.module}/../../infrastructure/istio/istio-cni-values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/infrastructure/istio/values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/infrastructure/istio/cni-values.yaml")
  ]
  depends_on = [helm_release.istiod]
  wait       = true
  timeout    = 300
  lifecycle { prevent_destroy = true }
}

resource "helm_release" "ztunnel" {
  name       = "ztunnel"
  namespace  = "istio-system"
  repository = "https://istio-release.storage.googleapis.com/charts"
  chart      = "ztunnel"
  version    = "1.30.3"
  values = [
    file("${path.module}/../../infrastructure/istio/ztunnel-values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/infrastructure/istio/values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/infrastructure/istio/ztunnel-values.yaml")
  ]
  depends_on = [helm_release.cni]
  wait       = true
  timeout    = 300
  lifecycle { prevent_destroy = true }
}
