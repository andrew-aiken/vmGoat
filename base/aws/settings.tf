terraform {
  required_version = ">= 1.0.0" # TODO: Set the version to be more specific

  # backend "local" {
  #   path = "/mnt/state/base.tfstate"
  # }
  backend "local" {}

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    local = {
      source = "hashicorp/local"
      version = "~> 2.5.0"
    }
    tls = {
      source = "hashicorp/tls"
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
      url       = "github.com/XXX" # TODO: Update with your repo URL
    }
  }
}
