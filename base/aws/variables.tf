variable "output_path" {
  type        = string
  default     = "/mnt/"
  description = "Path to where persistent data should be stored"
}

variable "allowlist" {
  type        = list(string)
  default     = []
  description = "description"
}
