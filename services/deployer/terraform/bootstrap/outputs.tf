output "locust_slave_count" {
  value = data.terraform_remote_state.provision.outputs.slave_instance_core
}

output "cluster" {
  value = {
    api_server_url  = rke_cluster.cluster.api_server_url
    kube_admin_user = rke_cluster.cluster.kube_admin_user

    client_cert = rke_cluster.cluster.client_cert
    client_key  = rke_cluster.cluster.client_key
    ca_crt      = rke_cluster.cluster.ca_crt
  }
}

output "nodes" {
  value = {
    locust_master_ip = data.terraform_remote_state.provision.outputs.locust_master_ip
    ssh_username     = data.terraform_remote_state.provision.outputs.ssh_username
    private_key      = data.terraform_remote_state.provision.outputs.private_key
  }
}

output "kubeconfig" {
  value = rke_cluster.cluster.kube_config_yaml
}
