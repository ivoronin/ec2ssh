# Security Group for Public Instance
resource "aws_security_group" "public_instance" {
  name        = "${local.name_prefix}-public-sg"
  description = "Security group for public E2E test instance"
  vpc_id      = aws_vpc.main.id

  # SSH from allowed CIDR (IPv4)
  ingress {
    description = "SSH from allowed CIDR"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.ssh_allowed_cidr]
  }

  # SSH from allowed CIDR (IPv6)
  ingress {
    description      = "SSH from allowed CIDR (IPv6)"
    from_port        = 22
    to_port          = 22
    protocol         = "tcp"
    ipv6_cidr_blocks = [var.ssh_allowed_ipv6_cidr]
  }

  # All egress (IPv4)
  egress {
    description = "All egress"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # All egress (IPv6)
  egress {
    description      = "All egress (IPv6)"
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    ipv6_cidr_blocks = ["::/0"]
  }

  tags = {
    Name = "${local.name_prefix}-public-sg"
  }
}

# Security Group for Private Instance
resource "aws_security_group" "private_instance" {
  name        = "${local.name_prefix}-private-sg"
  description = "Security group for private E2E test instance"
  vpc_id      = aws_vpc.main.id

  # SSH from EICE Security Group
  ingress {
    description     = "SSH from EICE"
    from_port       = 22
    to_port         = 22
    protocol        = "tcp"
    security_groups = [aws_security_group.eice.id]
  }

  # SSH from VPC CIDR (for SSM tunneling - traffic originates from instance)
  ingress {
    description = "SSH from VPC (SSM)"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }

  # SSH from VPC IPv6 CIDR
  ingress {
    description      = "SSH from VPC IPv6"
    from_port        = 22
    to_port          = 22
    protocol         = "tcp"
    ipv6_cidr_blocks = [aws_vpc.main.ipv6_cidr_block]
  }

  # All egress (IPv4)
  egress {
    description = "All egress"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # All egress (IPv6)
  egress {
    description      = "All egress (IPv6)"
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    ipv6_cidr_blocks = ["::/0"]
  }

  tags = {
    Name = "${local.name_prefix}-private-sg"
  }
}

# Security Group for EICE
resource "aws_security_group" "eice" {
  name        = "${local.name_prefix}-eice-sg"
  description = "Security group for EC2 Instance Connect Endpoint"
  vpc_id      = aws_vpc.main.id

  # EICE needs no ingress rules - it initiates connections

  # Egress to private subnet (SSH) - IPv4
  egress {
    description = "SSH to private subnet"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.private_subnet_cidr]
  }

  # Egress to private subnet (SSH) - IPv6
  egress {
    description      = "SSH to private subnet IPv6"
    from_port        = 22
    to_port          = 22
    protocol         = "tcp"
    ipv6_cidr_blocks = [cidrsubnet(aws_vpc.main.ipv6_cidr_block, 8, 2)]
  }

  tags = {
    Name = "${local.name_prefix}-eice-sg"
  }
}

# Security Group for VPC Endpoints (SSM)
resource "aws_security_group" "vpc_endpoints" {
  name        = "${local.name_prefix}-vpce-sg"
  description = "Security group for SSM VPC endpoints"
  vpc_id      = aws_vpc.main.id

  # HTTPS from VPC (for SSM API calls)
  ingress {
    description = "HTTPS from VPC"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }

  tags = {
    Name = "${local.name_prefix}-vpce-sg"
  }
}
