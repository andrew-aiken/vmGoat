variable "allow_list" {
  type        = list(string)
  default     = []
  description = "description"
}

variable "aws_region" {
  type        = string
  default     = "us-east-1"
  description = "description"
}

variable "aws_profile" {
  type        = string
  description = "description"
}
