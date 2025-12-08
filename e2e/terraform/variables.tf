variable "project_name" {
  description = "Project name for resource naming"
  type        = string
  default     = "ec2ssh-e2e"
}

variable "vpc_cidr" {
  description = "CIDR block for VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "public_subnet_cidr" {
  description = "CIDR block for public subnet"
  type        = string
  default     = "10.0.1.0/24"
}

variable "private_subnet_cidr" {
  description = "CIDR block for private subnet"
  type        = string
  default     = "10.0.2.0/24"
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}

variable "ssh_allowed_cidr" {
  description = "CIDR allowed for direct SSH access (IPv4)"
  type        = string
  default     = "0.0.0.0/0"
}

variable "ssh_allowed_ipv6_cidr" {
  description = "CIDR allowed for direct SSH access (IPv6)"
  type        = string
  default     = "::/0"
}
