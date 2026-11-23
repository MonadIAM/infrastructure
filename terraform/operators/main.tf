resource "helm_release" "cnpg" {
  name       = "cnpg"
  namespace  = "cnpg-system"
  repository = "https://cloudnative-pg.io/charts"
  chart      = "cloudnative-pg"
  version    = "0.29.0"
  values = [
    file("${path.module}/../../clusters/${var.profile}/infrastructure/postgresql/operator/values.yaml")
  ]
  wait    = true
  timeout = 600
  lifecycle { prevent_destroy = true }
}

resource "helm_release" "strimzi" {
  name      = "strimzi"
  namespace = "strimzi-system"
  chart     = "oci://quay.io/strimzi-helm/strimzi-kafka-operator"
  version   = "1.2.0"
  values = [
    file("${path.module}/../../clusters/${var.profile}/infrastructure/kafka/operator/values.yaml")
  ]
  wait    = true
  timeout = 600
  lifecycle { prevent_destroy = true }
}

resource "helm_release" "clickhouse_operator" {
  name       = "clickhouse-operator"
  namespace  = "clickhouse-operator"
  repository = "https://helm.altinity.com"
  chart      = "altinity-clickhouse-operator"
  version    = "0.27.3"
  values = [
    file("${path.module}/../../clusters/${var.profile}/infrastructure/clickhouse/operator/values.yaml")
  ]
  wait    = true
  timeout = 600
  lifecycle { prevent_destroy = true }
}
