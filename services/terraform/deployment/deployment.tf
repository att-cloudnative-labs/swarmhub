terraform {
  backend "s3" {
  }
  required_providers {
    aws        = "~> 2.48"
    kubernetes = "~> 1.10"
    null       = "~> 2.1"
  }
}

provider "aws" {
  region = var.bucket_region
}

locals {
  #locust_image        = "grubykarol/locust:0.12.0-python3.7-alpine3.10"
  locust_image        = "noclih/locust:0.0.1"
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
    "locust_prometheus.py" = "from itertools import chain\n\nfrom flask import make_response\nfrom locust import runners, stats, web\n\n\n@web.app.route(\"/metrics\")\ndef prometheus_metrics():\n    is_distributed = isinstance(\n        runners.locust_runner, runners.MasterLocustRunner)\n    if is_distributed:\n        slave_count = runners.locust_runner.slave_count\n    else:\n        slave_count = 0\n\n    if runners.locust_runner.host:\n        host = runners.locust_runner.host\n    elif len(runners.locust_runner.locust_classes) > 0:\n        host = runners.locust_runner.locust_classes[0].host\n    else:\n        host = None\n\n    state = 1\n    if runners.locust_runner.state != \"running\":\n        state = 0\n\n    rows = []\n    for s in stats.sort_stats(runners.locust_runner.request_stats):\n        rows.append(\"locust_request_count{{region=\\\"{}\\\", grid=\\\"{}\\\", test=\\\"{}\\\", endpoint=\\\"{}\\\", method=\\\"{}\\\"}} {}\\n\"\n                    \"locust_request_per_second{{region=\\\"{}\\\", grid=\\\"{}\\\", test=\\\"{}\\\", endpoint=\\\"{}\\\"}} {}\\n\"\n                    \"locust_failed_requests{{region=\\\"{}\\\", grid=\\\"{}\\\", test=\\\"{}\\\", endpoint=\\\"{}\\\", method=\\\"{}\\\"}} {}\\n\"\n                    \"locust_average_response{{region=\\\"{}\\\", grid=\\\"{}\\\", test=\\\"{}\\\", endpoint=\\\"{}\\\", method=\\\"{}\\\"}} {}\\n\"\n                    \"locust_average_content_length{{region=\\\"{}\\\", grid=\\\"{}\\\", test=\\\"{}\\\", endpoint=\\\"{}\\\", method=\\\"{}\\\"}} {}\\n\"\n                    \"locust_max_response_time{{region=\\\"{}\\\", grid=\\\"{}\\\", test=\\\"{}\\\", endpoint=\\\"{}\\\", method=\\\"{}\\\"}} {}\\n\"\n                    \"locust_running{{region=\\\"{}\\\", grid=\\\"{}\\\", test=\\\"{}\\\", site=\\\"{}\\\"}} {}\\n\"\n                    \"locust_workers{{region=\\\"{}\\\", grid=\\\"{}\\\", test=\\\"{}\\\", site=\\\"{}\\\"}} {}\\n\"\n                    \"locust_users{{region=\\\"{}\\\", grid=\\\"{}\\\", test=\\\"{}\\\", site=\\\"{}\\\"}} {}\\n\".format(\n                        '${data.terraform_remote_state.cluster.outputs.grid_region}',\n                        '${data.terraform_remote_state.cluster.outputs.grid_id}',\n                        '${var.test_name}',\n                        s.name,\n                        s.method,\n                        s.num_requests,\n                        '${data.terraform_remote_state.cluster.outputs.grid_region}',\n                        '${data.terraform_remote_state.cluster.outputs.grid_id}',\n                        '${var.test_name}',\n                        s.name,\n                        s.total_rps,\n                        '${data.terraform_remote_state.cluster.outputs.grid_region}',\n                        '${data.terraform_remote_state.cluster.outputs.grid_id}',\n                        '${var.test_name}',\n                        s.name,\n                        s.method,\n                        s.num_failures,\n                        '${data.terraform_remote_state.cluster.outputs.grid_region}',\n                        '${data.terraform_remote_state.cluster.outputs.grid_id}',\n                        '${var.test_name}',\n                        s.name,\n                        s.method,\n                        s.avg_response_time,\n                        '${data.terraform_remote_state.cluster.outputs.grid_region}',\n                        '${data.terraform_remote_state.cluster.outputs.grid_id}',\n                        '${var.test_name}',\n                        s.name,\n                        s.method,\n                        s.avg_content_length,\n                        '${data.terraform_remote_state.cluster.outputs.grid_region}',\n                        '${data.terraform_remote_state.cluster.outputs.grid_id}',\n                        '${var.test_name}',\n                        s.name,\n                        s.method,\n                        s.max_response_time,\n                        '${data.terraform_remote_state.cluster.outputs.grid_region}',\n                        '${data.terraform_remote_state.cluster.outputs.grid_id}',\n                        '${var.test_name}',\n                        host,\n                        state,\n                        '${data.terraform_remote_state.cluster.outputs.grid_region}',\n                        '${data.terraform_remote_state.cluster.outputs.grid_id}',\n                        '${var.test_name}',\n                        host,\n                        slave_count,\n                        '${data.terraform_remote_state.cluster.outputs.grid_region}',\n                        '${data.terraform_remote_state.cluster.outputs.grid_id}',\n                        '${var.test_name}',\n                        host,\n                        runners.locust_runner.user_count,\n                    )\n                    )\n\n    response = make_response(\"\".join(rows))\n    response.mimetype = \"text/plain; charset=utf-8'\"\n    response.content_type = \"text/plain; charset=utf-8'\"\n    response.headers[\"Content-Type\"] = \"text/plain; charset=utf-8'\"\n    return response\n"

    "locustfile.py"    = "${file("${path.module}/locustfile.py")}"
    "requirements.txt" = "${file("${path.module}/requirements.txt")}"
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
      service_name = "locust-master-svc"
      service_port = 80
    }

    rule {
      http {
        path {
          backend {
            service_name = "locust-master-svc"
            service_port = 8089
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
        container {
          image             = local.locust_image
          name              = "locust-master"
          image_pull_policy = "IfNotPresent"

          #env {
          #  name = "ATTACKED_HOST"
          #  value_from {
          #    config_map_key_ref {
          #      name = "locust-cm"
          #      key  = "ATTACKED_HOST"
          #    }
          #  }
          #}
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

          #env {
          #  name = "ATTACKED_HOST"
          #  value_from {
          #    config_map_key_ref {
          #      name = "locust-cm"
          #      key  = "ATTACKED_HOST"
          #    }
          #  }
          #}
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
    "prometheus.yml" = "global:\n  evaluation_interval: 1m\n  scrape_interval: 1m\n  scrape_timeout: 10s\n\nscrape_configs:\n- job_name: prometheus\n  static_configs:\n  - targets:\n    - localhost:9090\n\n- bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token\n  job_name: ${data.terraform_remote_state.cluster.outputs.grid_region}/${data.terraform_remote_state.cluster.outputs.grid_id}\n  kubernetes_sd_configs:\n  - role: endpoints\n    api_server: ${data.terraform_remote_state.cluster.outputs.cluster.api_server_url}\n    tls_config:\n      ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt\n      cert_file: /var/run/secrets/kubernetes.io/serviceaccount/cert.crt\n      key_file: /var/run/secrets/kubernetes.io/serviceaccount/key.crt\n      insecure_skip_verify: true\n    namespaces:\n      names:\n      - default\n\n  relabel_configs:\n  - action: keep\n    regex: true\n    source_labels:\n    - __meta_kubernetes_service_annotation_prometheus_io_scrape\n  - action: replace\n    regex: (https?)\n    source_labels:\n    - __meta_kubernetes_service_annotation_prometheus_io_scheme\n    target_label: __scheme__\n  - action: replace\n    regex: (.+)\n    source_labels:\n    - __meta_kubernetes_service_annotation_prometheus_io_path\n    target_label: __metrics_path__\n  - action: replace\n    regex: ([^:]+)(?::\\d+)?;(\\d+)\n    replacement: $1:$2\n    source_labels:\n    - __address__\n    - __meta_kubernetes_service_annotation_prometheus_io_port\n    target_label: __address__\n  - action: labelmap\n    regex: __meta_kubernetes_service_label_(.+)\n  - action: replace\n    source_labels:\n    - __meta_kubernetes_namespace\n    target_label: kubernetes_namespace\n  - action: replace\n    source_labels:\n    - __meta_kubernetes_service_name\n    target_label: kubernetes_name\n  - action: replace\n    source_labels:\n    - __meta_kubernetes_pod_node_name\n    target_label: kubernetes_node\n"
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
      node_port   = "30001"
    }
    selector = {
      app = "prometheus"
    }
    session_affinity = "None"
    type             = "NodePort"
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
          args  = ["--storage.tsdb.retention.time=15d", "--config.file=/etc/config/prometheus.yml", "--storage.tsdb.path=/data", "--web.console.libraries=/etc/prometheus/console_libraries", "--web.console.templates=/etc/prometheus/consoles", "--web.enable-lifecycle"]

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
#!/bin/sh
n=1
retry=300
until [ $n -ge $retry ]; do
    MASTER_STATUS=$(
        curl -s -o /dev/null -w "%%{http_code}" "http://${kubernetes_service.master-service.spec.0.cluster_ip}:8089/swarm" \
        -X POST -H "Content-Type: application/x-www-form-urlencoded" \
        -d "locust_count=${var.locust_count}&hatch_rate=${var.hatch_rate}"
        )
    if [ "$MASTER_STATUS" = "200"  ]; then
      RES=$(curl -s -w "\n%%{http_code}" "http://${kubernetes_service.master-service.spec.0.cluster_ip}:8089/stats/requests")
      STATUS=$(echo "$RES" | tail -n 1)
      if [ "$STATUS" = "200"  ]; then
        BODY=$(echo "$RES" | sed '$d')
        STATE=$(echo "$BODY" | python3 -c "import sys, json; print(json.load(sys.stdin)['state'])")
        if [ "$STATE" = "running" ];then
          exit 0
        else
          echo "Locust not in running state: ($STATE)"
        fi
      else
        echo "Failed to get /stats/requests: Status ($STATUS)"
      fi
    else
        echo "Locust master not ready: Status ($MASTER_STATUS)"
        echo "Attempt $n out of $retry..."
    fi
    n=$((n + 1))
    sleep 1
done
exit 1
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
