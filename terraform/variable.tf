variable "PROJECT_ID" {
  type        = string
  description = "ID of google project to deploy to."
}

variable "APP_NAME" {
  type        = string
  description = "Name of application"
  default     = "owlllm"
}

variable "NS" {
  type        = string
  description = "api"
}

variable "DB_HOST" {
  type        = string
  description = "Host to use in db connection string, default k8s db proxy"
  default     = "db-proxy.common"
}

