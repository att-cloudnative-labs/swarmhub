terraform {
  backend "s3" {
  }
}

data "terraform_remote_state" "provision" {
  backend = "s3"
  config = {
    bucket = var.bucket_tfstate
    region = var.bucket_region
    key    = var.tfstate_provision
  }
}

resource "rke_cluster" "cluster" {
  nodes {
    address = data.terraform_remote_state.provision.outputs.kube_master_ip
    user    = data.terraform_remote_state.provision.outputs.ssh_username
    ssh_key = data.terraform_remote_state.provision.outputs.private_key
    role    = ["controlplane", "etcd"]
  }
  nodes {
    address = data.terraform_remote_state.provision.outputs.locust_master_ip
    user    = data.terraform_remote_state.provision.outputs.ssh_username
    ssh_key = data.terraform_remote_state.provision.outputs.private_key
    role    = ["worker"]
  }
  dynamic nodes {
    for_each = data.terraform_remote_state.provision.outputs.locust_slave_ips[*]
    content {
      address = nodes.value
      user    = data.terraform_remote_state.provision.outputs.ssh_username
      ssh_key = data.terraform_remote_state.provision.outputs.private_key
      role    = ["worker"]
    }
  }
}

resource "local_file" "kube_cluster_yaml" {
  filename = "./kube_config_cluster.yml"
  content  = rke_cluster.cluster.kube_config_yaml
}
