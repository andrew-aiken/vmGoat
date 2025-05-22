resource "aws_instance" "this" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = "t3.small"
  subnet_id     = data.aws_subnets.public.ids[0]

  key_name = "vmGoat"

  metadata_options {
    http_endpoint          = "enabled"
    instance_metadata_tags = "disabled"
  }

  vpc_security_group_ids = [
    data.aws_security_group.this.id
  ]

  tags = {
    Name = "HelloWorld"
  }
}
