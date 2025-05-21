variable "aws_region" {
  type        = string
  default     = "us-east-1"
  description = "description"
}

variable "aws_profile" {
  type        = string
  default     = ""
  description = "description"
}

variable "allowlist" {
  type        = list(string)
  default     = []
  description = "description"
}
