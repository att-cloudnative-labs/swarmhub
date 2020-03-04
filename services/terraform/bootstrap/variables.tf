variable "bucket_tfstate" {
  type        = string
  description = "Bucket that stores tfstate"
}

variable "bucket_region" {
  type        = string
  description = "Region of the bucket"
}

variable "tfstate_provision" {
  type        = string
  description = "Key name of state file"
}
