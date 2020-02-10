variable "bucket_tfstate" {
  type        = string
  description = "Bucket that stores tfstate"
}

variable "bucket_region" {
  type        = string
  description = "Region of the bucket"
}

variable "tfstate_bootstrap" {
  type        = string
  description = "Key name of state file"
}

variable "stop_test" {
  type        = bool
  description = "Stop running test"
  default     = false
}

variable "hatch_rate" {
  type        = string
  description = "Hatch rate"
  default     = 0
}

variable "locust_count" {
  type        = string
  description = "Number of locust user to simulate"
  default     = 0

}
