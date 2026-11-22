resource "helm_release" "tempo" {
  name       = "tempo"
  namespace  = "monadiam-observability"
  repository = "https://grafana-community.github.io/helm-charts"
  chart      = "tempo"
  version    = "2.3.0"
  values = [
    file("${path.module}/../../infrastructure/tempo/values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/infrastructure/tempo/values.yaml")
  ]
  wait    = true
  timeout = 600
  lifecycle { prevent_destroy = true }
}
