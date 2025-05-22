resource "aws_route53_record" "gitoops" {
  zone_id = data.aws_route53_zone.public.zone_id
  name    = data.aws_route53_zone.public.name
  type    = "A"
  ttl     = "300"
  records = [aws_instance.gitoops.public_ip]
}

resource "tls_private_key" "access_key" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "aws_key_pair" "ssh_key" {
  key_name   = "gitoops"
  public_key = tls_private_key.access_key.public_key_openssh
}
