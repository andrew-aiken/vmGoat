terraform {
  backend "local" {
    path = "/mnt/state/base.tfstate"
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}


provider "aws" {
  region = var.aws_region
  shared_config_files      = ["/mnt/aws/config"]
  shared_credentials_files = ["/mnt/aws/credentials"]
  profile                  = var.aws_profile

  default_tags {
    tags = {
      Terraform = "true"
    }
  }
}
