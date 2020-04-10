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

variable "TLS_KEY_PATH" {
  type        = string
  description = "tls key path"
  default     = "/etc/tls/server.key"
}

variable "TLS_CRT_PATH" {
  type        = string
  description = "tls cert path"
  default     = "/etc/tls/server.crt"
}
