locals {
  services = toset([
    for path in fileset("${path.module}/../../clusters/${var.profile}/infrastructure/redis", "*.yaml") : trimsuffix(path, ".yaml") if path != "values.yaml"
  ])
}

resource "helm_release" "redis" {
  for_each  = local.services
  name      = "${each.key}-redis"
  namespace = "monadiam-data"
  chart     = "oci://registry-1.docker.io/bitnamicharts/redis"
  version   = "28.0.0"
  values = [
    file("${path.module}/../../clusters/${var.profile}/infrastructure/redis/values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/infrastructure/redis/${each.key}.yaml")
  ]
  wait    = true
  timeout = 600
  lifecycle { prevent_destroy = true }
}
