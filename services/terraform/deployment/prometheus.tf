resource "kubernetes_service_account" "prometheus-sa" {
  metadata {
    labels = {
      app = "prometheus"
    }
    name = "prometheus-server"
  }
}

data "kubernetes_secret" "prometheus-secret" {
  metadata {
    name      = "${kubernetes_service_account.prometheus-sa.default_secret_name}"
    namespace = "${kubernetes_service_account.prometheus-sa.metadata.0.namespace}"
  }
  depends_on = [
    kubernetes_service_account.prometheus-sa,
  ]
}

resource "kubernetes_config_map" "prometheus_server" {
  metadata {
    name = "prometheus-server"

    labels = {
      app = "prometheus"
    }
  }

  data = {
    "prometheus.yml" = templatefile("${path.module}/prometheus.yml.tmpl", {
      grid_region = data.terraform_remote_state.cluster.outputs.grid_region,
      grid_id = data.terraform_remote_state.cluster.outputs.grid_id,
      api_server_url = data.terraform_remote_state.cluster.outputs.cluster.api_server_url
    })
  }
}

resource "kubernetes_config_map" "prometheus_server_token" {
  metadata {
    name = "prometheus-server-token"

    labels = {
      app = "prometheus"
    }
  }

  data = {
    "ca.crt" = data.kubernetes_secret.prometheus-secret.data["ca.crt"]

    "cert.crt" = [
      for cert in data.terraform_remote_state.cluster.outputs.cluster.certificates :
      cert.certificate
      if cert.common_name == "kube-admin"
    ][0]

    "key.crt" = [
      for cert in data.terraform_remote_state.cluster.outputs.cluster.certificates :
      cert.key
      if cert.common_name == "kube-admin"
    ][0]

    "token" = data.kubernetes_secret.prometheus-secret.data["token"]

  }
}


resource "kubernetes_storage_class" "prometheus-sc" {
  metadata {
    name = "local-storage"
  }
  storage_provisioner = "kubernetes.io/no-provisioner"

}

resource "kubernetes_persistent_volume" "pv_server" {
  metadata {
    name = "pv-server"
    labels = {
      type = "local"
    }

    annotations = {
      "pv.kubernetes.io/bound-by-controller" = "yes"
    }
  }

  spec {
    capacity = {
      storage = "2Gi"
    }

    access_modes                     = ["ReadWriteOnce"]
    persistent_volume_reclaim_policy = "Retain"
    storage_class_name               = "local-storage"

    persistent_volume_source {
      local {
        path = "/tmp/pv_server"
      }
    }

    node_affinity {
      required {
        node_selector_term {
          match_expressions {
            key      = "node-role.kubernetes.io/worker"
            operator = "In"
            values   = ["true"]
          }
        }
      }
    }
  }
}

resource "kubernetes_persistent_volume_claim" "prometheus-pvc" {
  metadata {
    labels = {
      app = "prometheus"
    }
    name = "prometheus-server"
  }
  spec {
    access_modes       = ["ReadWriteOnce"]
    storage_class_name = "local-storage"
    resources {
      requests = {
        storage = "2Gi"
      }
    }
    volume_name = "pv-server"
  }
}

resource "kubernetes_cluster_role" "prometheus-cr" {
  metadata {
    labels = {
      app = "prometheus"
    }
    name = "prometheus-server"
  }

  rule {
    api_groups = [""]
    resources  = ["nodes", "nodes/proxy", "nodes/metrics", "services", "endpoints", "pods", "ingresses", "configmaps"]
    verbs      = ["get", "list", "watch"]
  }
  rule {
    api_groups = ["extensions"]
    resources  = ["ingresses/status", "ingresses"]
    verbs      = ["get", "list", "watch"]
  }
  rule {
    non_resource_urls = ["/metrics"]
    verbs             = ["get"]
  }
}

resource "kubernetes_cluster_role_binding" "prometheus-crb" {
  metadata {
    labels = {
      app = "prometheus"
    }
    name = "prometheus-server"
  }
  subject {
    kind = "ServiceAccount"
    name = "prometheus-server"
  }
  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "prometheus-server"
  }
}

resource "kubernetes_service" "prometheus-svc" {
  metadata {
    labels = {
      app = "prometheus"
    }
    name = "prometheus-server"
  }
  spec {
    port {
      name        = "http"
      port        = "80"
      protocol    = "TCP"
      target_port = "9090"
    }
    selector = {
      app = "prometheus"
    }
    session_affinity = "None"
  }
}

resource "kubernetes_deployment" "prometheus-deploy" {
  metadata {
    name = "prometheus-server"
    labels = {
      app = "prometheus"
    }
  }

  spec {
    replicas = 1
    selector {
      match_labels = {
        app = "prometheus"
      }
    }

    template {
      metadata {
        labels = {
          app = "prometheus"
        }
      }

      spec {
        node_selector = {
          app = "locust-master"
        }
        volume {
          name = "config-volume"
          config_map {
            name = "prometheus-server"
          }
        }

        volume {
          name = "token-config-volume"
          config_map {
            name = "prometheus-server-token"
          }
        }

        volume {
          name = "storage-volume"
          persistent_volume_claim {
            claim_name = "prometheus-server"
          }
        }

        container {
          name  = "prometheus-server-configmap-reload"
          image = "jimmidyson/configmap-reload:v0.2.2"
          args  = ["--volume-dir=/etc/config", "--webhook-url=http://127.0.0.1:9090/-/reload"]

          volume_mount {
            name       = "config-volume"
            read_only  = true
            mount_path = "/etc/config"
          }

          image_pull_policy = "IfNotPresent"
        }

        container {
          name  = "prometheus-server"
          image = "prom/prometheus:v2.13.1"
          args  = [
            "--storage.tsdb.retention.time=15d",
            "--config.file=/etc/config/prometheus.yml",
            "--storage.tsdb.path=/data",
            "--web.console.libraries=/etc/prometheus/console_libraries",
            "--web.console.templates=/etc/prometheus/consoles",
            "--web.enable-lifecycle",
            "--web.external-url=http://localhost:9090/prometheus/",
            "--web.route-prefix=/"
          ]

          port {
            container_port = 9090
          }

          volume_mount {
            name       = "config-volume"
            mount_path = "/etc/config"
          }

          volume_mount {
            name       = "token-config-volume"
            mount_path = "/var/run/secrets/kubernetes.io/serviceaccount"
          }

          volume_mount {
            name       = "storage-volume"
            mount_path = "/data"
          }

          liveness_probe {
            http_get {
              path = "/-/healthy"
              port = "9090"
            }

            initial_delay_seconds = 30
            timeout_seconds       = 30
            success_threshold     = 1
            failure_threshold     = 3
          }

          readiness_probe {
            http_get {
              path = "/-/ready"
              port = "9090"
            }

            initial_delay_seconds = 30
            timeout_seconds       = 30
            success_threshold     = 1
            failure_threshold     = 3
          }

          image_pull_policy = "IfNotPresent"
        }

        security_context {
          fs_group        = "0"
          run_as_group    = "0"
          run_as_non_root = "false"
          run_as_user     = "0"
        }

        termination_grace_period_seconds = 600
        service_account_name             = "prometheus-server"
      }
    }
  }
}