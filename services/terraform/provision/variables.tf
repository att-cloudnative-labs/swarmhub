variable "grid_id" {
  type        = string
  description = "Name of the grid"
}

variable "cloud_provider" {
  type        = string
  description = "Type of provider"
}

variable "grid_region" {
  type        = string
  description = "Region to launch grid in"
}

variable "master_instance_type" {
  default     = ""
  type        = string
  description = "Instance type to be used for master node"
}

variable "slave_instance_type" {
  default     = ""
  type        = string
  description = "Instance type to be used for slave node"
}

variable "slave_instance_count" {
  default     = 0
  type        = string
  description = "Total k8 worker nodes use to deploy locust slave"
}

variable "slave_instance_core" {
  default     = 0
  type        = string
  description = "Total k8 worker nodes' core"
}
