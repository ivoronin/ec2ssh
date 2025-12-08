# VPC with dual-stack (IPv4 + IPv6)
resource "aws_vpc" "main" {
  cidr_block                       = var.vpc_cidr
  assign_generated_ipv6_cidr_block = true
  enable_dns_hostnames             = true
  enable_dns_support               = true

  tags = {
    Name = "${local.name_prefix}-vpc"
  }
}

# Internet Gateway
resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "${local.name_prefix}-igw"
  }
}

# Egress-only Internet Gateway for IPv6 (private subnet)
resource "aws_egress_only_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "${local.name_prefix}-eigw"
  }
}

# Public Subnet (dual-stack)
resource "aws_subnet" "public" {
  vpc_id                          = aws_vpc.main.id
  cidr_block                      = var.public_subnet_cidr
  ipv6_cidr_block                 = cidrsubnet(aws_vpc.main.ipv6_cidr_block, 8, 1)
  map_public_ip_on_launch         = true
  assign_ipv6_address_on_creation = true
  availability_zone               = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "${local.name_prefix}-public-subnet"
  }
}

# Private Subnet (dual-stack)
resource "aws_subnet" "private" {
  vpc_id                          = aws_vpc.main.id
  cidr_block                      = var.private_subnet_cidr
  ipv6_cidr_block                 = cidrsubnet(aws_vpc.main.ipv6_cidr_block, 8, 2)
  map_public_ip_on_launch         = false
  assign_ipv6_address_on_creation = true
  availability_zone               = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "${local.name_prefix}-private-subnet"
  }
}

# Public Route Table
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  # IPv4 route to IGW
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  # IPv6 route to IGW
  route {
    ipv6_cidr_block = "::/0"
    gateway_id      = aws_internet_gateway.main.id
  }

  tags = {
    Name = "${local.name_prefix}-public-rt"
  }
}

# Private Route Table
# No NAT Gateway needed - SSM connectivity via VPC Endpoints
resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id

  # IPv6 route to Egress-only IGW (free, unlike NAT Gateway)
  route {
    ipv6_cidr_block        = "::/0"
    egress_only_gateway_id = aws_egress_only_internet_gateway.main.id
  }

  tags = {
    Name = "${local.name_prefix}-private-rt"
  }
}

# Route Table Associations
resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.public.id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table_association" "private" {
  subnet_id      = aws_subnet.private.id
  route_table_id = aws_route_table.private.id
}
