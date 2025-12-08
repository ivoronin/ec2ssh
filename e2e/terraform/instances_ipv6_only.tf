# IPv6-only instance

resource "aws_instance" "ipv6_only" {
  ami                    = data.aws_ami.al2023.id
  instance_type          = var.instance_type
  subnet_id              = aws_subnet.ipv6_only.id
  vpc_security_group_ids = [aws_security_group.ipv6_only_instance.id]
  iam_instance_profile   = aws_iam_instance_profile.ec2_ssm.name
  ipv6_address_count     = 1

  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 2
  }

  root_block_device {
    volume_type           = "gp3"
    volume_size           = 30
    encrypted             = true
    delete_on_termination = true
  }

  tags = {
    Name = "${local.name_prefix}-ipv6-only"
  }
}
