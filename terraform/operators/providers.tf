provider "helm" {
  kubernetes = {
    config_path    = var.kubeconfig
    config_context = var.context
  }
}
