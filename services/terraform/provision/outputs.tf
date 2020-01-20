output "locust_slave_count" {
  value = var.slave_instance_core
}

output "nodes" {
  value = {
    locust_master_ip = module.nodes.locust_master_ip
    ssh_username     = module.nodes.ssh_username
    private_key      = module.nodes.private_key
  }

}
