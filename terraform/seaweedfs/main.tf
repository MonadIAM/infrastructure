resource "helm_release" "seaweedfs_main" {
  name       = "seaweedfs-main"
  namespace  = "monadiam-data"
  repository = "https://seaweedfs.github.io/seaweedfs/helm"
  chart      = "seaweedfs"
  version    = "4.48.0"
  values = [
    file("${path.module}/../../infrastructure/seaweedfs/values.yaml"),
    file("${path.module}/../../clusters/${var.profile}/infrastructure/seaweedfs/values.yaml")
  ]
  wait    = true
  timeout = 600
  lifecycle { prevent_destroy = true }
}
