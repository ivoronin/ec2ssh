# EC2 SSH Connection Tool
![GitHub release (with filter)](https://img.shields.io/github/v/release/ivoronin/ec2ssh)
![GitHub last commit (branch)](https://img.shields.io/github/last-commit/ivoronin/ec2ssh/main)
![GitHub Workflow Status (with event)](https://img.shields.io/github/actions/workflow/status/ivoronin/ec2ssh/goreleaser.yml)
![GitHub top language](https://img.shields.io/github/languages/top/ivoronin/ec2ssh)

## Description
This CLI tool eases secure SSH connections to AWS EC2 instances. It automatically retrieves the instance's IP address, sends the SSH public key using AWS EC2 Instance Connect, and initiates an SSH connection directly or through the AWS EC2 Instance Connect Endpoint.

![](demo/demo.webp)

# Features
- Identifies EC2 instances by ID, DNS name, IP address, or name tag.
- Automatically fetches EC2 instance's public or private IP addresses.
- Sends SSH public key to instances using AWS EC2 Instance Connect.
- Tunnels SSH connections through AWS EC2 Instance Connect Endpoint.

## Usage
```
Usage: ec2ssh [ec2ssh options] [ssh arguments] destination [command [argument ...]]

Connect to an EC2 instance directly using SSH or via the EC2 Instance Connect
Endpoint (EICE), by the instance ID, private, public, or IPv6 address, private
DNS name, or name tag, using ephemeral SSH keys.

  Example - Connect to an instance using the instance ID:
     $ ec2ssh -l ec2-user i-0123456789abcdef0

  Example - Connect to an instance using a name tag with the public IP address:
     $ ec2ssh -p 2222 --address-type public ec2-user@app01

  Example - Connect to an instance using its private DNS name via an EICE tunnel:
     $ ec2ssh --use-eice ip-10-0-0-1

Options:
  --region <string>
     Use the specified AWS region (env AWS_REGION, AWS_DEFAULT_REGION).
     Defaults to using the AWS SDK configuration.

  --profile <string>
     Use the specified AWS profile (env AWS_PROFILE).
     Defaults to using the AWS SDK configuration.

  --use-eice
     Use EC2 Instance Connect Endpoint (EICE) to connect to the instance.
     Default is false. Ignores --address-type, private address is always used.

  --eice-id <string>
     Specifies the EC2 Instance Connect Endpoint (EICE) ID to use.
     Defaults to autodetection based on the instance's VPC and subnet.
     Automatically implies --use-eice.

  --destination-type <id|private_ip|public_ip|ipv6|private_dns|name_tag>
     Specify the destination type for instance search.
     Defaults to automatically detecting the type based on the destination.
     First matched instance will be user for connection.

  --address-type <private|public|ipv6>
     Specify the address type for connecting to the instance.
     Defaults to use the first available address from the list: private, public, ipv6.

  --no-send-keys
     Do not send SSH keys to the instance using EC2 Instance Connect.

  --debug
     Enable debug logging.

  ssh arguments
     Specify arguments to pass to SSH.

  destination
     Specify the destination for connection. Can be one of: instance ID,
     private, public or IPv6 IP address, private DNS name, or name tag.
```

## Configuration & AWS Credentials
- **AWS Region, Access Key ID and Secret**: Configure the AWS SDK using `aws configure` or set the `AWS_DEFAULT_REGION`, `AWS_ACCESS_KEY_ID`, and `AWS_SECRET_ACCESS_KEY` environment variables to the corresponding values.
