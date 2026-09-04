output "api_endpoint" {
  value       = "http://localhost/api/v1"
  description = "Ingress URL for high-concurrency Go API"
}

output "grafana_dashboard_url" {
  value       = "http://localhost:3000"
  description = "Grafana dashboard URL (via NodePort 30000)"
}

output "prometheus_url" {
  value       = "http://localhost:9090"
  description = "Prometheus UI URL (via NodePort 30090)"
}

output "hpa_status_command" {
  value       = "kubectl get hpa -n ecommerce"
  description = "Command to observe dynamic horizontal pod scaling under 10k RPS"
}
