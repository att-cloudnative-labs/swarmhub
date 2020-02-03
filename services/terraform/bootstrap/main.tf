terraform {
  backend "s3" {
  }
  required_providers {
    local = "~> 1.4"
    rke   = "~> 0.14"
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
    labels = {
      app = "locust-master"
    }
  }
  dynamic nodes {
    for_each = data.terraform_remote_state.provision.outputs.locust_slave_ips[*]
    content {
      address = nodes.value
      user    = data.terraform_remote_state.provision.outputs.ssh_username
      ssh_key = data.terraform_remote_state.provision.outputs.private_key
      role    = ["worker"]
      labels = {
        app = "locust-slave"
      }
    }
  }
  services_kubelet {
    extra_binds = ["/etc/config:/etc/config:rshared", "/tmp/pv_server:/tmp/pv_server:rshared", "/tmp/data:/tmp/data:rshared", "/data:/data:rshared", "/var/run/secrets/kubernetes.io/serviceaccount:/var/run/secrets/kubernetes.io/serviceaccount:rshared"]
  }
}

resource "local_file" "kube_cluster_yaml" {
  filename = "./kube_config_cluster.yml"
  content  = rke_cluster.cluster.kube_config_yaml
}