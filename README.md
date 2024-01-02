# EC2 SSH Connection Tool
## Description
This CLI tool eases secure SSH connections to AWS EC2 instances. It automatically retrieves the instance's IP address, sends the SSH public key using AWS EC2 Instance Connect, and initiates an SSH connection directly or through the AWS EC2 Instance Connect Endpoint.

# Features
- Identifies EC2 instances by ID, DNS name, IP address, or name tag.
- Automatically fetches EC2 instance's public or private IP addresses.
- Sends SSH public key to instances using AWS EC2 Instance Connect.
- Tunnels SSH connections through AWS EC2 Instance Connect Endpoint.

## Usage
```
ec2ssh [ec2ssh flags] [ssh flags] destination [command [argument ...]]
```
- `--region`: AWS region in which the instance search is to be performed.
- `--profile`: AWS configuration profile to use.
- `--use-eice`: Connect using the EC2 Instance Connect Endpoint. Default is false. Conflicts with `--use-public-ip`.
- `--eice-id`: EC2 Instance Connect Endpoint Id. Implies `--use-eice`. Default is to guess based on instance's VPC and subnet.
- `--destination-type`: Interpret destination as instance `id`, `private_ip`, `public_ip`, `private_dns` or `name_tag`. Default is `auto`.
- `--address-type`: Connect to instance using speicified address type: `private`, `public` or `ipv6`. Default is `private`.
- `--no-send-keys`: Do not send keys using EC2 Instance Connect. Default is false.
- `destination`: Can be an instance ID (e.g., `i-1234567890abcdef0`), private DNS name (e.g., `ip-172-31-32-101`), private or public IP address, or a Name tag value.
- `ssh flags`: Additional flags to pass to the SSH command.
- `[command [argument ...]]`: Optional command to execute on the remote instance.

## Examples
- Connect to an instance using its ID: `ec2ssh i-1234567890abcdef0`
- Connect using an instance name: `ec2ssh -l ubuntu my-instance-name-tag-value`
- Connect using an AWS EC2 Instance Connect Endpoint tunneling: `ec2ssh -l ubuntu --use-eice ip-172-16-45-80`

## Configuration & AWS Credentials
- **AWS Region, Access Key ID and Secret**: Configure the AWS SDK using `aws configure` or set the `AWS_DEFAULT_REGION`, `AWS_ACCESS_KEY_ID`, and `AWS_SECRET_ACCESS_KEY` environment variables to the corresponding values.
