terraform {
  backend "s3" {
  }
  required_providers {
    aws        = "~> 2.48"
    kubernetes = "~> 1.10"
    null       = "~> 2.1"
    tls        = "~> 2.1"
  }
}

provider "aws" {
  region = var.bucket_region
}

locals {
  locust_image        = "noclih/locust:0.0.1"
  locust_proxy_image  = "jenglamlow/locust-proxy:0.1"
  locust_scripts_path = "/locust"
}

data "terraform_remote_state" "cluster" {
  backend = "s3"
  config = {
    bucket = var.bucket_tfstate
    region = var.bucket_region
    key    = var.tfstate_bootstrap
  }
}

provider "kubernetes" {
  load_config_file = false

  host     = data.terraform_remote_state.cluster.outputs.cluster.api_server_url
  username = data.terraform_remote_state.cluster.outputs.cluster.kube_admin_user

  client_certificate     = data.terraform_remote_state.cluster.outputs.cluster.client_cert
  client_key             = data.terraform_remote_state.cluster.outputs.cluster.client_key
  cluster_ca_certificate = data.terraform_remote_state.cluster.outputs.cluster.ca_crt
}

resource "kubernetes_config_map" "scripts_cm" {
  metadata {
    name = "scripts-cm"
  }

  data = {
    "locust_prometheus.py" = templatefile("${path.module}/locust_prometheus.py.tmpl", {
      grid_region = data.terraform_remote_state.cluster.outputs.grid_region,
      grid_id = data.terraform_remote_state.cluster.outputs.grid_id,
      test_id = var.test_id,
    })
    "locustfile.py"    = file("${path.module}/locustfile.py")
    "requirements.txt" = file("${path.module}/requirements.txt")
  }
}

resource "kubernetes_secret" "proxy-secret" {
  metadata {
    name = "proxy-secret"
  }
  data = {
    "jwt" = file("/etc/jwt/jwt")
  }
}

resource "kubernetes_ingress" "ingress" {
  metadata {
    name = "locust"
    annotations = {
      "ingress.kubernetes.io/rewrite-target"    = "/"
    }
  }

  spec {
    rule {
      http {
        path {
          backend {
            service_name = "locust-master-svc"
            service_port = 8001
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "master-service" {
  metadata {
    name = "locust-master-svc"
    labels = {
      role = "locust-master"
    }
    annotations = {
      "prometheus.io/scrape" = true
      "prometheus.io/port"   = 8089
    }
  }
  spec {
    selector = {
      role = "locust-master"
    }
    port {
      name = "comm"
      port = 5557
    }
    port {
      name = "comm-plus-1"
      port = 5558
    }
    port {
      name = "web-ui"
      port = 8089
    }
    port {
      name = "proxy"
      port = 8001
    }
  }
}

resource "kubernetes_deployment" "master-deployment" {
  metadata {
    name = "locust-master"
    labels = {
      role = "locust-master"
    }
  }

  spec {
    replicas = 1

    selector {
      match_labels = {
        role = "locust-master"
      }
    }

    strategy {
      rolling_update {
        max_surge       = 1
        max_unavailable = 1
      }
    }

    template {
      metadata {
        labels = {
          role = "locust-master"
        }
        annotations = {
          "prometheus.io/scrape" = true
          "prometheus.io/port"   = 8089
        }
      }

      spec {
        node_selector = {
          app = "locust-master"
        }

        # Locust Master
        container {
          image             = local.locust_image
          name              = "locust-master"
          image_pull_policy = "IfNotPresent"

          env {
            name  = "LOCUST_MODE"
            value = "MASTER"
          }

          env {
            name  = "LOCUST_OPTS"
            value = "--print-stats"
          }

          volume_mount {
            mount_path = local.locust_scripts_path
            name       = "locust-scripts"
          }

          port {
            name           = "comm"
            container_port = 5557
          }
          port {
            name           = "comm-plus-1"
            container_port = 5558
          }
          port {
            name           = "web-ui"
            container_port = 8089
          }
        }

        # Locust Proxy
        container {
          name              = "locust-proxy"
          image             = local.locust_proxy_image
          image_pull_policy = "Always"

          volume_mount {
            mount_path = "/jwt"
            sub_path   = "jwt"
            name       = "jwt"
          }

          port {
            name           = "web-ui-proxy"
            container_port = 8001
          }
        }

        toleration {
          effect   = "NoSchedule"
          operator = "Exists"
        }

        termination_grace_period_seconds = 30

        # Volumes
        volume {
          name = "locust-scripts"
          config_map {
            name = "scripts-cm"
          }
        }
        volume {
          name = "jwt"
          secret {
            secret_name = "proxy-secret"
          }
        }
      }
    }
  }
}

resource "kubernetes_deployment" "slave-deployment" {
  metadata {
    name = "locust-slave"
    labels = {
      role = "locust-slave"
    }
    annotations = {
      "deployment.kubernetes.io/revision" = "1"
    }
  }

  spec {
    replicas = data.terraform_remote_state.cluster.outputs.locust_slave_count

    selector {
      match_labels = {
        role = "locust-slave"
      }
    }

    template {
      metadata {
        labels = {
          role = "locust-slave"
        }
        annotations = {
          "prometheus.io/scrape" = true
          "prometheus.io/port"   = 8089
        }
      }

      spec {
        node_selector = {
          app = "locust-slave"
        }
        container {
          image             = local.locust_image
          name              = "locust-slave"
          image_pull_policy = "IfNotPresent"

          env {
            name  = "LOCUST_MODE"
            value = "SLAVE"
          }
          env {
            name  = "LOCUST_MASTER"
            value = "false"
          }
          env {
            name  = "LOCUST_OPTS"
            value = "--print-stats"
          }

          env {
            name  = "LOCUST_MASTER_HOST"
            value = kubernetes_service.master-service.spec.0.cluster_ip
          }

          env {
            name  = "LOCUST_MASTER_PORT"
            value = 5557
          }

          volume_mount {
            mount_path = local.locust_scripts_path
            name       = "locust-scripts"
          }
        }
        toleration {
          effect   = "NoSchedule"
          operator = "Exists"
        }

        termination_grace_period_seconds = 30
        volume {
          name = "locust-scripts"
          config_map {
            name = "scripts-cm"
          }
        }
      }
    }
  }
}

resource "null_resource" "swarm" {
  triggers = {
    locust_count = var.locust_count
    hatch_rate   = var.hatch_rate
  }

  connection {
    type        = "ssh"
    host        = data.terraform_remote_state.cluster.outputs.nodes.locust_master_ip
    user        = data.terraform_remote_state.cluster.outputs.nodes.ssh_username
    agent       = false
    private_key = data.terraform_remote_state.cluster.outputs.nodes.private_key
  }

  depends_on = [kubernetes_deployment.slave-deployment, kubernetes_deployment.master-deployment]
  provisioner "file" {
    content = templatefile("${path.module}/swarm.tmpl", {
      locust_count = var.locust_count,
      hatch_rate   = var.hatch_rate,
      locust_master_ip = kubernetes_service.master-service.spec.0.cluster_ip
    })
    destination = "/tmp/swarm.sh"
  }
  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/swarm.sh",
      "/tmp/swarm.sh",
    ]
  }
}

resource "null_resource" "stop_test" {
  count = var.stop_test ? 1 : 0

  connection {
    type        = "ssh"
    host        = data.terraform_remote_state.cluster.outputs.nodes.locust_master_ip
    user        = data.terraform_remote_state.cluster.outputs.nodes.ssh_username
    agent       = false
    private_key = data.terraform_remote_state.cluster.outputs.nodes.private_key
  }

  depends_on = [kubernetes_deployment.slave-deployment, kubernetes_deployment.master-deployment]
  provisioner "remote-exec" {
    inline = [<<EOT
      curl "http://${kubernetes_service.master-service.spec.0.cluster_ip}:8089/stop"
      EOT
    ]
  }
}
