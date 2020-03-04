variable "kube_master_instance_type" {
  type        = string
  description = "k8 master node intance type to use"
}

variable "grid_id" {
  type        = string
  description = "Name of the grid"
}

variable "grid_region" {
  type        = string
  description = "Region to launch grid in"
}

variable "master_instance_type" {
  type        = string
  description = "Instance type to be used for master node"
  default     = ""
}

variable "slave_instance_type" {
  type        = string
  description = "Instance type to be used for slave node"
  default     = ""
}

variable "slave_instance_count" {
  type        = number
  description = "Total k8 slave nodes"
  default     = 0
}

variable "slave_instance_core" {
  default     = 0
  type        = string
  description = "Total k8 worker nodes' core"
}

variable "ttl" {
  default     = 0
  type        = string
  description = "Time-to-live of the grid"
}
