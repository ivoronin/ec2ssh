output "eice_id" {
  value = aws_ec2_instance_connect_endpoint.main.id
}

output "public_id" {
  value = aws_instance.public.id
}

output "public_ip" {
  value = aws_instance.public.public_ip
}

output "public_ipv6" {
  value = aws_instance.public.ipv6_addresses[0]
}

output "private_id" {
  value = aws_instance.private.id
}

output "private_ip" {
  value = aws_instance.private.private_ip
}

output "private_ipv6" {
  value = aws_instance.private.ipv6_addresses[0]
}

output "public_name" {
  description = "Name tag of public instance"
  value       = aws_instance.public.tags["Name"]
}

output "private_name" {
  description = "Name tag of private instance"
  value       = aws_instance.private.tags["Name"]
}
