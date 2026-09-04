variable "kubeconfig_path" {
  type        = string
  default     = "~/.kube/config"
  description = "Path to the kubeconfig file"
}

variable "kubeconfig_context" {
  type        = string
  default     = ""
  description = "Kubernetes context to use"
}

variable "namespace" {
  type        = string
  default     = "ecommerce"
  description = "Kubernetes namespace for e-commerce resources"
}

variable "api_replicas" {
  type        = number
  default     = 3
  description = "Initial number of Go API replicas before HPA auto-scaling"
}

variable "api_max_replicas" {
  type        = number
  default     = 10
  description = "Maximum number of Go API replicas under 10k RPS load"
}

variable "worker_replicas" {
  type        = number
  default     = 2
  description = "Number of Kafka consumer batch workers"
}

variable "environment" {
  type        = string
  default     = "production"
  description = "Deployment environment tag"
}
