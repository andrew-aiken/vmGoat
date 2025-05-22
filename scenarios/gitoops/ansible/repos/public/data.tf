data "aws_vpc" "vpc" {
  filter {
    name   = "tag:Name"
    values = ["thm"]
  }
}

data "aws_subnet" "subnet" {
  vpc_id            = data.aws_vpc.vpc.id

  filter {
    name   = "tag:Name"
    values = ["thm"]
  }
}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

data "aws_route53_zone" "public" {
  name         = var.dns_zone_domain
  private_zone = false
}
