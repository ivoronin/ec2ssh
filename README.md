# EC2 SSH Connection Tool
## Description
This CLI tool facilitates secure SSH connections to AWS EC2 instances. It automatically retrieves the instance's IP address, sends the SSH public key using AWS EC2 Instance Connect, and initiates an SSH connection.

## Features
- Flexible instance identification (by ID, DNS name, IP address, or name tag).
- Automatic retrieval of the EC2 instance's IP address.
- Supports sending the SSH public key to the instance using AWS EC2 Instance Connect.

## Usage
```
ec2ssh [-l login_user] [other ssh flags] destination [command [argument ...]]
```
- `-l login_user`: Specify the login user for the SSH connection (default: `ec2-user`).
- `destination`: Can be an instance ID (e.g., `i-1234567890abcdef0`), private DNS name (e.g., `ip-172-31-32-101`), private or public IP address, or a Name tag value.
- `other ssh flags`: Additional flags to pass to the SSH command.
- `[command [argument ...]]`: Optional command to execute on the remote instance.

## Examples
- Connect to an instance using its ID: `ec2ssh i-1234567890abcdef0`
- Connect using an instance name: `ec2ssh -l ubuntu my-instance-name-tag-value`

## Configuration
- **AWS Region, Access Key Id and Secret**: Set the `AWS_DEFAULT_REGION`, `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables to corresponding values.
- **SSH Public Key Path**: Set the `SSH_PUBLIC_KEY_PATH` environment variable to specify the path to your SSH public key file. If not set, the default path is `~/.ssh/id_rsa.pub`.
