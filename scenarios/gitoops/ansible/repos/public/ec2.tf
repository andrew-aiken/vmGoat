resource "aws_security_group" "gitoops" {
  name        = "gitoops"
  description = "Allow traffic to vmGoat gitoops server"
  vpc_id      = data.aws_vpc.vpc.id

  ingress {
    description = "HTTPS"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "SSH into server"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }

  tags = {
    Name = "gitoops"
  }
}

resource "aws_instance" "gitoops" {
  ami                         = data.aws_ami.ubuntu.image_id
  instance_type               = "t3.medium"
  subnet_id                   = data.aws_subnet.subnet.id
  vpc_security_group_ids      = [aws_security_group.gitoops.id]
  associate_public_ip_address = true
  private_ip                  = "10.1.0.123"
  key_name                    = aws_key_pair.ssh_key.id

  tags = {
    Name = "gitoops"
  }
}
