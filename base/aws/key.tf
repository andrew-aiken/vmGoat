resource "tls_private_key" "rsa" {
  algorithm = "RSA"
  rsa_bits  = 4096
}


resource "aws_key_pair" "this" {
  key_name   = "vmGoat"
  public_key = tls_private_key.rsa.public_key_openssh
}


resource "local_file" "private_key" {
  content         = tls_private_key.rsa.private_key_pem
  filename        = "/mnt/ssh/id_rsa"
  file_permission = "0600"
}

resource "local_file" "public_key" {
  content         = tls_private_key.rsa.public_key_openssh
  filename        = "/mnt/ssh/id_rsa.pub"
  file_permission = "0600"
}
