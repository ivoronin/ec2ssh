# Dedicated VPC for IPv6-only testing
# Separate VPC to avoid EICE quota limits (1 per VPC)

resource "aws_vpc" "ipv6_only" {
  cidr_block                       = "10.99.0.0/16" # Minimal IPv4 (required by AWS)
  assign_generated_ipv6_cidr_block = true
  enable_dns_hostnames             = true
  enable_dns_support               = true

  tags = {
    Name = "${local.name_prefix}-ipv6-only-vpc"
  }
}

# Internet Gateway for public IPv6
resource "aws_internet_gateway" "ipv6_only" {
  vpc_id = aws_vpc.ipv6_only.id

  tags = {
    Name = "${local.name_prefix}-ipv6-only-igw"
  }
}

# IPv6-Only Subnet (public - directly reachable via IPv6)
resource "aws_subnet" "ipv6_only" {
  vpc_id                                         = aws_vpc.ipv6_only.id
  ipv6_cidr_block                                = cidrsubnet(aws_vpc.ipv6_only.ipv6_cidr_block, 8, 0)
  assign_ipv6_address_on_creation                = true
  ipv6_native                                    = true
  enable_dns64                                   = true
  enable_resource_name_dns_aaaa_record_on_launch = true
  availability_zone                              = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "${local.name_prefix}-ipv6-only-subnet"
  }
}

# Route Table for IPv6-only subnet (public via IGW)
resource "aws_route_table" "ipv6_only" {
  vpc_id = aws_vpc.ipv6_only.id

  route {
    ipv6_cidr_block = "::/0"
    gateway_id      = aws_internet_gateway.ipv6_only.id
  }

  tags = {
    Name = "${local.name_prefix}-ipv6-only-rt"
  }
}

resource "aws_route_table_association" "ipv6_only" {
  subnet_id      = aws_subnet.ipv6_only.id
  route_table_id = aws_route_table.ipv6_only.id
}
