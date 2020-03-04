output "locust_master_ip" {
  value = kubernetes_service.master-service.spec.0.cluster_ip
}
