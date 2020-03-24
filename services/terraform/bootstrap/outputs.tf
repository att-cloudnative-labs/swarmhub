output "locust_slave_count" {
  value = data.terraform_remote_state.provision.outputs.slave_instance_core
}

output "cluster" {
  depends_on = [rke_cluster.cluster]
  value = {
    api_server_url  = rke_cluster.cluster.api_server_url
    kube_admin_user = rke_cluster.cluster.kube_admin_user

    client_cert  = rke_cluster.cluster.client_cert
    client_key   = rke_cluster.cluster.client_key
    ca_crt       = rke_cluster.cluster.ca_crt
    certificates = rke_cluster.cluster.certificates
  }
  sensitive = true
}

output "nodes" {
  value = {
    locust_master_ip = data.terraform_remote_state.provision.outputs.locust_master_ip
    ssh_username     = data.terraform_remote_state.provision.outputs.ssh_username
    private_key      = data.terraform_remote_state.provision.outputs.private_key
  }
}

output "kubeconfig" {
  value     = rke_cluster.cluster.kube_config_yaml
  sensitive = true
}

output "provider" {
  value = data.terraform_remote_state.provision.outputs.provider
}

output "grid_id" {
  value = data.terraform_remote_state.provision.outputs.grid_id
}

output "grid_region" {
  value = data.terraform_remote_state.provision.outputs.grid_region
}

