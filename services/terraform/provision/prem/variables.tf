variable "slave_instance_core" {
  default     = 0
  type        = string
  description = "Total k8 worker nodes' core"
}

variable "private_key" {
  type        = string
  description = "Private key for on prem nodes"
}

variable "ssh_username" {
  type        = string
  description = "SSH username for on prem nodes"
}

variable "kube_master_ip" {
  type        = string
  description = "IP of prem kube master node"
}

variable "locust_master_ip" {
  type        = string
  description = "IP of prem locust master node"
}

variable "locust_slave_ips" {
  type        = list
  description = "List of ips of prem nodes"
}
