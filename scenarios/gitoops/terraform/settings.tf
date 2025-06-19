terraform {
  required_version = ">= 1.12.0"

  backend "local" {}

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}


provider "aws" {
  default_tags {
    tags = {
      scenario  = "test"
      source    = "mvGoat"
      terraform = "true"
      url       = "https://github.com/andrew-aiken/vmGoat"
    }
  }
}
