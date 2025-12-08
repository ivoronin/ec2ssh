# Endpoints for IPv6-only VPC

# EICE for IPv6-only VPC
resource "aws_ec2_instance_connect_endpoint" "ipv6_only" {
  subnet_id          = aws_subnet.ipv6_only.id
  security_group_ids = [aws_security_group.eice_ipv6_only.id]
  preserve_client_ip = false

  tags = {
    Name = "${local.name_prefix}-eice-ipv6-only"
  }
}

# VPC Endpoint for S3 (Gateway) - for SSM agent updates
resource "aws_vpc_endpoint" "s3_ipv6_only" {
  vpc_id            = aws_vpc.ipv6_only.id
  service_name      = "com.amazonaws.${data.aws_region.current.region}.s3"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = [aws_route_table.ipv6_only.id]

  tags = {
    Name = "${local.name_prefix}-ipv6-only-vpce-s3"
  }
}
