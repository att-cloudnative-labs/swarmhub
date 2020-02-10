terraform {
  backend "s3" {
  }
}

provider "aws" {
  region = var.bucket_region
}

locals {
  locust_image        = "grubykarol/locust:0.12.0-python3.7-alpine3.10"
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

resource "kubernetes_config_map" "scripts-cm" {
  metadata {
    name = "scripts-cm"
  }

  data = {
    "locustfile.py"        = "${file("${path.module}/locustfile.py")}"
    "locust_prometheus.py" = "${file("${path.module}/locust_prometheus.py")}"
  }
}

resource "kubernetes_config_map" "locust-cm" {
  metadata {
    name = "locust-cm"
  }
  data = {
    ATTACKED_HOST = "http://locust-master:8089"
  }
}

resource "kubernetes_ingress" "ingress" {
  metadata {
    name = "locust"
    annotations = {
      "ingress.kubernetes.io/rewrite-target" = "/"
      "ingress.kubernetes.io/ssl-redirect"   = "false"
    }
  }

  spec {
    backend {
      service_name = "locust-master"
      service_port = 80
    }

    rule {
      http {
        path {
          backend {
            service_name = "locust-master"
            service_port = 8089
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "master-service" {
  metadata {
    name = "locust-master"
    labels = {
      role = "locust-master"
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
      }

      spec {
        container {
          image             = local.locust_image
          name              = "locust-master"
          image_pull_policy = "IfNotPresent"

          env {
            name = "ATTACKED_HOST"
            value_from {
              config_map_key_ref {
                name = "locust-cm"
                key  = "ATTACKED_HOST"
              }
            }
          }
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




          resources {}
          termination_message_path = "/dev/termination-log"
        }
        toleration {
          effect   = "NoSchedule"
          operator = "Exists"
        }

        dns_policy     = "ClusterFirst"
        restart_policy = "Always"
        security_context {}
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

    /*strategy {
      rolling_update {
        max_surge       = 2
        max_unavailable = 0
      }
    }*/

    template {
      metadata {
        labels = {
          role = "locust-slave"
        }
      }

      spec {
        container {
          image             = local.locust_image
          name              = "locust-slave"
          image_pull_policy = "IfNotPresent"

          env {
            name = "ATTACKED_HOST"
            value_from {
              config_map_key_ref {
                name = "locust-cm"
                key  = "ATTACKED_HOST"
              }
            }
          }
          env {
            name  = "LOCUST_MODE"
            value = "SLAVE"
          }
          env {
            name  = "LOCUST_MASTER"
            value = "locust-master"
          }
          env {
            name  = "LOCUST_OPTS"
            value = "--print-stats"
          }

          volume_mount {
            mount_path = local.locust_scripts_path
            name       = "locust-scripts"
          }

          resources {}
          termination_message_path = "/dev/termination-log"
        }
        toleration {
          effect   = "NoSchedule"
          operator = "Exists"
        }

        dns_policy     = "ClusterFirst"
        restart_policy = "Always"
        security_context {}
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
  provisioner "remote-exec" {
    inline = [
      <<EOT
      curl "http://${kubernetes_service.master-service.spec.0.cluster_ip}:8089/swarm" \
      -X POST -H "Content-Type: application/x-www-form-urlencoded" \
      -d "locust_count=${var.locust_count}&hatch_rate=${var.hatch_rate}"
      EOT
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
