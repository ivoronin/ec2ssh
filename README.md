# ec2ssh

![GitHub release (with filter)](https://img.shields.io/github/v/release/ivoronin/ec2ssh)
[![Go Report Card](https://goreportcard.com/badge/github.com/ivoronin/ec2ssh)](https://goreportcard.com/report/github.com/ivoronin/ec2ssh)
![GitHub last commit (branch)](https://img.shields.io/github/last-commit/ivoronin/ec2ssh/main)
![GitHub Workflow Status (with event)](https://img.shields.io/github/actions/workflow/status/ivoronin/ec2ssh/goreleaser.yml)
![GitHub top language](https://img.shields.io/github/languages/top/ivoronin/ec2ssh)

**SSH into EC2 instances the easy way.** No more juggling instance IDs, IP lookups, and key management.

```bash
ec2ssh my-app-server
```

That's it. You're connected.

![](demo/demo.webp)

## Why ec2ssh?

Connecting to EC2 instances the traditional way is painful:

```bash
# The old way: 5 commands, mass frustration
INSTANCE_ID=$(aws ec2 describe-instances --filters "Name=tag:Name,Values=my-app" \
  --query 'Reservations[0].Instances[0].InstanceId' --output text)
IP=$(aws ec2 describe-instances --instance-ids $INSTANCE_ID \
  --query 'Reservations[0].Instances[0].PrivateIpAddress' --output text)
ssh-keygen -t ed25519 -f /tmp/key -N ""
aws ec2-instance-connect send-ssh-public-key --instance-id $INSTANCE_ID \
  --instance-os-user ec2-user --ssh-public-key file:///tmp/key.pub
ssh -i /tmp/key ec2-user@$IP  # Quick! Before the 60-second key expires!
```

**ec2ssh** reduces this to a single command:

```bash
ec2ssh my-app
```

| Feature | Raw AWS CLI | ec2ssh |
|---------|-------------|--------|
| Commands needed | 5+ | 1 |
| Instance lookup by name | Complex `--query` syntax | Just use the name |
| SSH key management | Manual generate/push/cleanup | Automatic ephemeral keys |
| Private instances (EICE) | WebSocket setup, manual signing | `--use-eice` flag |
| IP auto-detection | Manual query + decision | Smart priority selection |
| SSH option passthrough | Separate command | Full compatibility |

## Features

- **Smart Instance Discovery** - Find instances by ID, name tag, private/public IP, IPv6, or DNS name
- **Zero Key Management** - Generates ephemeral ed25519 keys per session, auto-cleaned after use
- **Private Instance Access** - One flag (`--use-eice`) for EC2 Instance Connect Endpoint tunneling
- **Full SSH Compatibility** - All your favorite SSH options work: `-L`, `-D`, `-J`, `-o`, and more
- **Quick Instance Listing** - See all your instances with `--list`

## Installation

### Using Go

```bash
go install github.com/ivoronin/ec2ssh@latest
```

### From Releases

Download the latest binary for your platform from [GitHub Releases](https://github.com/ivoronin/ec2ssh/releases):

- **macOS**: `ec2ssh_Darwin_x86_64.tar.gz` or `ec2ssh_Darwin_arm64.tar.gz`
- **Linux**: `ec2ssh_Linux_x86_64.tar.gz` or `ec2ssh_Linux_arm64.tar.gz`
- **Windows**: `ec2ssh_Windows_x86_64.zip` or `ec2ssh_Windows_arm64.zip`

## Quick Start

```bash
# Connect by instance name tag
ec2ssh my-app-server

# Connect by instance ID
ec2ssh i-0123456789abcdef0

# Connect to a private instance via EICE tunnel
ec2ssh --use-eice my-private-server

# List all instances in the region
ec2ssh --list

# Use with standard SSH options
ec2ssh -L 8080:localhost:8080 my-app-server
```

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

  Example - Use any SSH options and arguments as usual:
     $ ec2ssh --use-eice -L 8888:127.0.0.1:8888 -N -i ~/.ssh/id_rsa_alt -o VisualHostKey=Yes app01

Options:
  --region <string>
     Use the specified AWS region (env AWS_REGION, AWS_DEFAULT_REGION).
     Defaults to using the AWS SDK configuration.

  --profile <string>
     Use the specified AWS profile (env AWS_PROFILE).
     Defaults to using the AWS SDK configuration.

  --list
     List instances in the region and exit.

  --list-columns <columns>
     Specify columns to display in the list output.
     Defaults to ID,NAME,STATE,PRIVATE-IP,PUBLIC-IP
     Available columns: ID,NAME,STATE,TYPE,PRIVATE-IP,PUBLIC-IP,IPV6,PRIVATE-DNS,PUBLIC-DNS

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
     First matched instance will be used for connection.

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

## Configuration

### AWS Credentials

ec2ssh uses the standard AWS SDK credential chain. Configure using any of:

```bash
# Option 1: AWS CLI configuration
aws configure

# Option 2: Environment variables
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_DEFAULT_REGION=us-east-1

# Option 3: AWS Profile
export AWS_PROFILE=my-profile
# or use --profile flag
ec2ssh --profile my-profile my-instance
```

### Required IAM Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances",
        "ec2-instance-connect:SendSSHPublicKey"
      ],
      "Resource": "*"
    }
  ]
}
```

For private instance access via EICE (`--use-eice`), add:

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

## Security

- **Ephemeral Keys**: ec2ssh generates a fresh ed25519 keypair for each session, stored in a temporary directory and automatically cleaned up
- **60-Second Window**: Keys pushed via EC2 Instance Connect are only valid for 60 seconds on the instance
- **EICE Tunnel Signing**: WebSocket tunnels use AWS Signature V4 with 60-second expiry
- **No Persistent Keys**: No SSH keys are stored permanently by ec2ssh

## License

GPL-3.0 - See [LICENSE](LICENSE) for details.
