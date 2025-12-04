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
Usage: ec2ssh [intent] [options] [user@]destination [command]
       ec2sftp [options] [user@]destination[:path]
       ec2scp [options] source target
       ec2list [options]

Intents (first argument or inferred from binary name ec2sftp/ec2scp/ec2list):
  --ssh (default), --sftp, --scp, --list, --help

AWS Options:
  --region <region>       AWS region (default: SDK config)
  --profile <profile>     AWS profile (default: SDK config)

Connection Options:
  --use-eice              Use EC2 Instance Connect Endpoint (default: false)
  --eice-id <id>          EICE ID (implies --use-eice, default: autodetect by VPC/subnet)
  --destination-type <t>  How to interpret destination (default: auto)
                          Values: id|private_ip|public_ip|ipv6|private_dns|name_tag
  --address-type <type>   Address for connection (default: auto)
                          Values: private|public|ipv6
  --no-send-keys          Skip sending SSH keys via EC2 Instance Connect (default: false)

List Options:
  --list-columns <cols>   Columns to display
                          Default: ID,NAME,STATE,PRIVATE-IP,PUBLIC-IP
                          Available: ID,NAME,STATE,TYPE,AZ,PRIVATE-IP,
                                     PUBLIC-IP,IPV6,PRIVATE-DNS,PUBLIC-DNS

Other:
  --debug                 Enable debug logging (default: false)

Examples:
  ec2ssh ec2-user@i-0123456789abcdef0
  ec2ssh --use-eice -L 8080:localhost:80 ubuntu@my-web-server
  ec2sftp -P 2222 user@app01:/var/log
  ec2scp -r --region us-west-2 ./logs admin@10.0.1.5:/backup/
  ec2list --profile prod --list-columns ID,NAME,STATE

All standard ssh/sftp/scp options are supported and passed through to the
underlying command.
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
