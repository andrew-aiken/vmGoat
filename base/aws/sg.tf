resource "aws_security_group" "this" {
  name        = "vmGoat-ingress"
  description = "Allow ingress from the allowlist"
  vpc_id      = aws_vpc.this.id

  tags = {
    Name = "vmGoat-ingress"
  }
}

resource "aws_vpc_security_group_ingress_rule" "this" {
  for_each = toset(var.allowlist)

  security_group_id = aws_security_group.this.id
  cidr_ipv4         = "${each.value}/32"
  ip_protocol       = "-1"
}

resource "aws_vpc_security_group_egress_rule" "egress" {
  security_group_id = aws_security_group.this.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
}
