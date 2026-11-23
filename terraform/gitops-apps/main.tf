resource "kubernetes_manifest" "project" {
  manifest = yamldecode(file("${path.module}/../../clusters/${var.profile}/infrastructure/argocd/project.yaml"))
  lifecycle { prevent_destroy = true }
}

resource "kubernetes_manifest" "root" {
  manifest   = yamldecode(file("${path.module}/../../clusters/${var.profile}/infrastructure/argocd/root.yaml"))
  depends_on = [kubernetes_manifest.project]
  lifecycle { prevent_destroy = true }
}
