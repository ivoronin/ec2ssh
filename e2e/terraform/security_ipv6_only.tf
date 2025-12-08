# Security groups for IPv6-only VPC

# Security Group for EICE in IPv6-only VPC
resource "aws_security_group" "eice_ipv6_only" {
  name        = "${local.name_prefix}-eice-ipv6-only-sg"
  description = "Security group for EICE in IPv6-only VPC"
  vpc_id      = aws_vpc.ipv6_only.id

  egress {
    description      = "SSH to IPv6-only subnet"
    from_port        = 22
    to_port          = 22
    protocol         = "tcp"
    ipv6_cidr_blocks = [aws_subnet.ipv6_only.ipv6_cidr_block]
  }

  tags = {
    Name = "${local.name_prefix}-eice-ipv6-only-sg"
  }
}

# Security Group for IPv6-only instance
resource "aws_security_group" "ipv6_only_instance" {
  name        = "${local.name_prefix}-ipv6-only-instance-sg"
  description = "Security group for IPv6-only instance"
  vpc_id      = aws_vpc.ipv6_only.id

  # SSH from allowed IPv6 CIDR (direct public access)
  ingress {
    description      = "SSH from allowed IPv6 CIDR"
    from_port        = 22
    to_port          = 22
    protocol         = "tcp"
    ipv6_cidr_blocks = [var.ssh_allowed_ipv6_cidr]
  }

  ingress {
    description     = "SSH from EICE"
    from_port       = 22
    to_port         = 22
    protocol        = "tcp"
    security_groups = [aws_security_group.eice_ipv6_only.id]
  }

  ingress {
    description      = "SSH from VPC IPv6 (for SSM)"
    from_port        = 22
    to_port          = 22
    protocol         = "tcp"
    ipv6_cidr_blocks = [aws_vpc.ipv6_only.ipv6_cidr_block]
  }

  egress {
    description      = "All IPv6 egress"
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    ipv6_cidr_blocks = ["::/0"]
  }

  tags = {
    Name = "${local.name_prefix}-ipv6-only-instance-sg"
  }
}
