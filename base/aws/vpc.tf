resource "aws_vpc" "this" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "vmGoat"
  }
}


resource "aws_internet_gateway" "this" {
  vpc_id = aws_vpc.this.id

  tags = {
    Name = "vmGoat"
  }
}


resource "aws_subnet" "public_subnet_a" {
  availability_zone       = "${var.aws_region}a"
  cidr_block              = cidrsubnet(aws_vpc.this.cidr_block, 8, 0)
  vpc_id                  = aws_vpc.this.id
  map_public_ip_on_launch = true

  tags = {
    Name = "vmGoat-public-a"
  }
}

resource "aws_subnet" "public_subnet_b" {
  availability_zone       = "${var.aws_region}b"
  cidr_block              = cidrsubnet(aws_vpc.this.cidr_block, 8, 1)
  vpc_id                  = aws_vpc.this.id
  map_public_ip_on_launch = true

  tags = {
    Name = "vmGoat-public-b"
  }
}


resource "aws_subnet" "private_subnet_a" {
  availability_zone       = "${var.aws_region}a"
  cidr_block              = cidrsubnet(aws_vpc.this.cidr_block, 8, 128)
  vpc_id                  = aws_vpc.this.id
  map_public_ip_on_launch = false

  tags = {
    Name = "vmGoat-private-a"
  }
}

resource "aws_subnet" "private_subnet_b" {
  availability_zone       = "${var.aws_region}b"
  cidr_block              = cidrsubnet(aws_vpc.this.cidr_block, 8, 129)
  vpc_id                  = aws_vpc.this.id
  map_public_ip_on_launch = false

  tags = {
    Name = "vmGoat-private-b"
  }
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.this.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.this.id
  }

  tags = {
    Name = "vmGoat-public"
  }
}

resource "aws_route_table" "private" {
  vpc_id = aws_vpc.this.id

  tags = {
    Name = "vmGoat-private"
  }
}


resource "aws_route_table_association" "public_subnet_a" {
  subnet_id      = aws_subnet.public_subnet_a.id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table_association" "public_subnet_b" {
  subnet_id      = aws_subnet.public_subnet_b.id
  route_table_id = aws_route_table.public.id
}


resource "aws_route_table_association" "private_subnet_a" {
  subnet_id      = aws_subnet.private_subnet_a.id
  route_table_id = aws_route_table.private.id
}

resource "aws_route_table_association" "private_subnet_b" {
  subnet_id      = aws_subnet.private_subnet_b.id
  route_table_id = aws_route_table.private.id
}
