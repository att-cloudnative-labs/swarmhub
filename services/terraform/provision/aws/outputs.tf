output "private_key" {
  value = tls_private_key.node-key.private_key_pem
}

output "ssh_username" {
  value = "ubuntu"
}

output "kube_master_ip" {
  value = aws_instance.rke-node-master[0].public_dns
}

output "locust_master_ip" {
  value = aws_instance.rke-node-slave-locust-master[0].public_dns
}

output "locust_slave_ips" {
  value = aws_instance.rke-node-slave-locust-slave[*].public_dns
}

output "slave_instance_core" {
  value = var.slave_instance_core
}

output "ttl" {
  value = var.ttl
}

output "provider" {
  value = "aws"
}

output "grid_id" {
  value = var.grid_id
}

output "grid_region" {
  value = var.grid_region
}
