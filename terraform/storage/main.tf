resource "helm_release" "rawfile" {
  name             = "rawfile-localpv"
  namespace        = "openebs"
  create_namespace = true
  repository       = "https://openebs.github.io/rawfile-localpv"
  chart            = "rawfile-localpv"
  version          = "0.15.3"
  values = [
    file("${path.module}/../../infrastructure/rawfile/values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/infrastructure/rawfile/values.yaml")
  ]
  wait    = true
  timeout = 300
  lifecycle { prevent_destroy = true }
}
