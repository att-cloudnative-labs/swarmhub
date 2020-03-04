provider "aws" {
  region = var.grid_region
}

terraform {
  backend "s3" {
  }
  required_providers {
    aws = "~> 2.48"
    tls = "~> 2.1"
  }
}
