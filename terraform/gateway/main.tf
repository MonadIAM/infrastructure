resource "helm_release" "gateway" {
  name           = "gateway"
  namespace      = "istio-ingress"
  chart          = "${path.module}/../../clusters/${var.profile}/infrastructure/gateway-api/chart"
  take_ownership = true
  set = [
    {
      name  = "applicationsIP"
      value = var.applications_ip
      type  = "string"
    }
  ]
  wait    = true
  timeout = 300
  lifecycle { prevent_destroy = true }
}
