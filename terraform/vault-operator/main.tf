resource "helm_release" "vault_secrets_operator" {
  name       = "vault-secrets-operator"
  namespace  = "vault-secrets-operator"
  repository = "https://helm.releases.hashicorp.com"
  chart      = "vault-secrets-operator"
  version    = "1.5.0"
  values = [
    file("${path.module}/../../clusters/${var.profile}/infrastructure/vault/operator/values.yaml")
  ]
  wait    = true
  timeout = 300
  lifecycle { prevent_destroy = true }
}
