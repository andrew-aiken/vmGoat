terraform {
  required_version = ">= 1.12.0"

  backend "local" {}

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.1.0"
    }
  }
}


provider "aws" {
  default_tags {
    tags = {
      scenario  = "base"
      source    = "mvGoat"
      terraform = "true"
      url       = "https://github.com/andrew-aiken/vmGoat"
    }
  }
}
