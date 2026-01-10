# ec2ssh

SSH to EC2 instances by Name tag or instance ID without manual IP lookup

[![CI](https://github.com/ivoronin/ec2ssh/actions/workflows/test.yml/badge.svg)](https://github.com/ivoronin/ec2ssh/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/ivoronin/ec2ssh)](https://github.com/ivoronin/ec2ssh/releases)

[Overview](#overview) · [Features](#features) · [Installation](#installation) · [Usage](#usage) · [Configuration](#configuration) · [Requirements](#requirements) · [License](#license)

```bash
# Before: look up instance IP and push ephemeral SSH key manually
aws ec2 describe-instances --filters "Name=tag:Name,Values=my-web-server" \
  --query "Reservations[].Instances[].[InstanceId,PublicIpAddress]" --output text
aws ec2-instance-connect send-ssh-public-key --instance-id i-0123456789abcdef0 \
  --instance-os-user ec2-user --ssh-public-key file://~/.ssh/id_ed25519.pub
ssh ec2-user@203.0.113.42  # key valid for 60 seconds

# After: connect by Name tag or instance ID directly
ec2ssh my-web-server
```

## Overview

ec2ssh resolves EC2 instances by Name tag, IP address, or instance ID, then generates an ephemeral ed25519 keypair and pushes the public key via EC2 Instance Connect API. The key is valid for 60 seconds on the instance. For private instances, ec2ssh tunnels through EC2 Instance Connect Endpoint (EICE) or SSM Session Manager without requiring bastion hosts or security group changes.

## Features

- Connects using Name tag, instance ID, private/public IP, IPv6, or private DNS
- Ephemeral ed25519 keys with 60-second TTL via EC2 Instance Connect API
- EICE tunneling for private instances (auto-discovers endpoint by VPC/subnet)
- SSM Session Manager tunneling and direct shell access
- SSM RunCommand execution with configurable timeout
- Full SSH/SCP/SFTP option passthrough (-L, -R, -J, -o, etc.)
- Instance listing with customizable columns
- Single Go binary with no runtime dependencies

## Installation

### Homebrew

```bash
brew install ivoronin/ivoronin/ec2ssh
```

### GitHub Releases

Download from [Releases](https://github.com/ivoronin/ec2ssh/releases).

### Symlinks

The binary auto-detects mode based on invocation name. Create symlinks for standalone commands:

```bash
ln -s ec2ssh ec2scp
ln -s ec2ssh ec2sftp
ln -s ec2ssh ec2ssm
ln -s ec2ssh ec2list
```

## Usage

### SSH

```bash
ec2ssh my-server                              # Connect by Name tag
ec2ssh i-0123456789abcdef0                    # Connect by instance ID
ec2ssh 10.0.1.50                              # Connect by private IP
ec2ssh ec2-user@my-server                     # Specify username
ec2ssh my-server uptime                       # Run command and exit
```

### SCP

```bash
ec2scp ./local-file.txt ec2-user@my-server:/tmp/
ec2scp -r ec2-user@my-server:/var/log/ ./logs/
ec2scp --region us-west-2 ./data admin@10.0.1.5:/backup/
```

### SFTP

```bash
ec2sftp my-server
ec2sftp ec2-user@my-server:/var/log
```

### Private Instances via EICE

```bash
ec2ssh --use-eice my-private-server           # Auto-discovers EICE endpoint
ec2ssh --eice-id eice-0abc123 my-server       # Specify EICE endpoint
ec2scp --use-eice ./file admin@my-server:/tmp/
```

### Private Instances via SSM

```bash
ec2ssh --use-ssm my-private-server            # SSH over SSM tunnel
ec2scp --use-ssm ./file admin@my-server:/tmp/
```

### SSM Shell (No SSH)

```bash
ec2ssm my-server                              # Interactive shell via SSM
ec2ssm i-0123456789abcdef0 whoami             # Run command via SSM RunCommand
ec2ssm --timeout 5m my-server ./long-script.sh
```

### Instance Listing

```bash
ec2list                                       # List all instances
ec2list --profile prod                        # Use specific AWS profile
ec2list --list-columns ID,NAME,STATE,AZ       # Custom columns
```

Available columns: `ID`, `NAME`, `STATE`, `TYPE`, `AZ`, `PRIVATE-IP`, `PUBLIC-IP`, `IPV6`, `PRIVATE-DNS`, `PUBLIC-DNS`

### SSH Options Passthrough

All standard SSH options pass through unchanged:

```bash
ec2ssh -L 3306:my-rds.cluster.us-east-1.rds.amazonaws.com:3306 my-server  # Local port forward
ec2ssh -R 8080:localhost:3000 my-server       # Remote port forward
ec2ssh -J bastion my-private-server           # Jump host
ec2ssh -o StrictHostKeyChecking=no my-server  # Custom SSH options
```

### Command Reference

```
Usage: ec2ssh [options] [user@]destination [command [args...]]
       ec2scp [options] source target
       ec2sftp [options] [user@]destination[:path]
       ec2ssm [options] destination [command [args...]]
       ec2list [options]

AWS Options:
  --region <region>       AWS region (default: SDK config)
  --profile <profile>     AWS profile (default: SDK config)

Connection Options:
  --use-eice              Use EC2 Instance Connect Endpoint
  --use-ssm               Use SSM Session Manager for tunneling
  --eice-id <id>          EICE ID (implies --use-eice, default: autodetect)
  --destination-type <t>  How to interpret destination (default: auto)
                          Values: id, private_ip, public_ip, ipv6, private_dns, name_tag
  --address-type <type>   Address for connection (default: auto)
                          Values: private, public, ipv6
  --no-send-keys          Skip EC2 Instance Connect key push

List Options:
  --list-columns <cols>   Columns to display (default: ID,NAME,STATE,PRIVATE-IP,PUBLIC-IP)

SSM Command Options:
  --timeout <duration>    Timeout for command completion (default: 60s)

Other:
  --debug                 Enable debug logging
  --help, --version       Show help or version
```

## Configuration

### AWS Credentials

ec2ssh uses the standard AWS SDK credential chain:

```bash
# AWS CLI configuration
aws configure

# Environment variables
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
export AWS_DEFAULT_REGION=us-east-1

# AWS profile
export AWS_PROFILE=my-profile
# or
ec2ssh --profile my-profile my-server
```

## Requirements

### Runtime Dependencies

- OpenSSH client (`ssh`, `scp`, `sftp`, `ssh-keygen`)

### IAM Permissions

Basic usage (SSH/SCP/SFTP):

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "ec2:DescribeInstances",
      "ec2-instance-connect:SendSSHPublicKey"
    ],
    "Resource": "*"
  }]
}
```

EICE tunneling (`--use-eice`):

```json
{
  "Effect": "Allow",
  "Action": [
    "ec2:DescribeInstanceConnectEndpoints",
    "ec2-instance-connect:OpenTunnel"
  ],
  "Resource": "*"
}
```

SSM access (`--use-ssm` or `ec2ssm`):

```json
{
  "Effect": "Allow",
  "Action": [
    "ssm:StartSession",
    "ssm:TerminateSession",
    "ssm:SendCommand",
    "ssm:GetCommandInvocation"
  ],
  "Resource": "*"
}
```

### Target Instance Requirements

- **Direct SSH**: Network connectivity to instance, SSH port open
- **EICE tunneling**: EC2 Instance Connect Endpoint in the VPC
- **SSM access**: SSM Agent installed and running on instance

## License

[GPL-3.0](LICENSE)
