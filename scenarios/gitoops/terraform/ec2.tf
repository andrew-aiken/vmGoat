resource "aws_instance" "this" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = "t3.medium"
  subnet_id     = data.aws_subnets.public.ids[0]

  key_name = "vmGoat"

  metadata_options {
    http_endpoint          = "enabled"
    instance_metadata_tags = "disabled"
  }

  vpc_security_group_ids = [
    data.aws_security_group.this.id
  ]

  user_data = <<-EOF
    #!/bin/bash
    echo 'BUCKET_NAME=${aws_s3_bucket.this.bucket}' > /vars.env
  EOF

  tags = {
    Name = "GitOops"
  }
}
