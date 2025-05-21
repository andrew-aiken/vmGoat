terraform {
  required_version = ">= 1.0.0" # TODO: Set the version to be more specific

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
  profile                  = var.aws_profile
  region                   = var.aws_region
  shared_config_files      = ["/mnt/aws/config"]
  shared_credentials_files = ["/mnt/aws/credentials"]

  default_tags {
    tags = {
      scenario  = "base"
      source    = "mvGoat"
      terraform = "true"
      url       = "github.com/XXX" # TODO: Update with your repo URL
    }
  }
}
