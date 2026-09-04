# 1. Namespace Creation
resource "kubernetes_namespace" "ecommerce" {
  metadata {
    name = var.namespace
    labels = {
      "environment"                = var.environment
      "app.kubernetes.io/part-of"  = "ecommerce-platform"
    }
  }
}

# 2. Ingress NGINX Controller via Helm
resource "helm_release" "ingress_nginx" {
  name             = "ingress-nginx"
  repository       = "https://kubernetes.github.io/ingress-nginx"
  chart            = "ingress-nginx"
  version          = "4.10.0"
  namespace        = "ingress-nginx"
  create_namespace = true

  set {
    name  = "controller.hostPort.enabled"
    value = "true"
  }

  set {
    name  = "controller.service.type"
    value = "NodePort"
  }

  set {
    name  = "controller.config.worker-processes"
    value = "auto"
  }

  set {
    name  = "controller.config.max-worker-connections"
    value = "65536"
  }

  set {
    name  = "controller.config.upstream-keepalive-connections"
    value = "300"
  }
}

# 3. Apply Core E-Commerce K8s Stack (Postgres, Redis, Kafka, API, Worker, HPA, Telemetry)
# Uses null_resource / local-exec for deterministic manifest application
resource "null_resource" "k8s_manifests" {
  depends_on = [
    kubernetes_namespace.ecommerce,
    helm_release.ingress_nginx
  ]

  triggers = {
    manifest_hash = filemd5("${path.module}/../k8s/04-backend-api.yaml")
  }

  provisioner "local-exec" {
    command = "kubectl apply -f ${path.module}/../k8s/"
  }
}
