terraform {
  backend "s3" {
  }
  required_providers {
    aws = "~> 2.48"
    tls = "~> 2.1"
  }
}

module "nodes" {
  source                      = "./aws"
  grid_region                 = var.grid_region
  grid_id                     = var.grid_id
  locust_master_instance_type = var.master_instance_type
  locust_slave_instance_type  = var.slave_instance_type
  locust_slave_instance_count = var.slave_instance_count
}

resource "local_file" "key" {
  filename = "./key"
  content  = module.nodes.private_key
}
