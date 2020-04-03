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
    role    = ["controlplane", "etcd", "worker"]
    labels = {
      app = "k8s-master"
    }
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

  # Monitoring node_selector is not supported in terraform-provider-rke v0.14.1. Will upgrade to the latest v1 version once it is stable released.
  monitoring {
    provider = "metrics-server"
  }

  # Limitation: coredns-autoscaler pod can't be nodeselect
    node_selector = {
      app = "k8s-master"
    }
    provider = "coredns"
  }

  # Limitation: default-http-backend pod can't be nodeselect
  ingress {
    node_selector = {
      app = "locust-master"
    }
    provider = "nginx"
  }

  services_kubelet {
    extra_binds = [
      "/etc/config:/etc/config:rshared",
      "/tmp/pv_server:/tmp/pv_server:rshared",
      "/tmp/data:/tmp/data:rshared", "/data:/data:rshared",
      "/var/run/secrets/kubernetes.io/serviceaccount:/var/run/secrets/kubernetes.io/serviceaccount:rshared"
    ]
  }
}

resource "local_file" "kube_cluster_yaml" {
  filename = "./kube_config_cluster.yml"
  content  = rke_cluster.cluster.kube_config_yaml
}
