provider "aws" {
  region = var.grid_region
}

terraform {
  backend "s3" {
  }
}
