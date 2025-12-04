# ec2ssh

![GitHub release (with filter)](https://img.shields.io/github/v/release/ivoronin/ec2ssh)
[![Go Report Card](https://goreportcard.com/badge/github.com/ivoronin/ec2ssh)](https://goreportcard.com/report/github.com/ivoronin/ec2ssh)
![GitHub last commit (branch)](https://img.shields.io/github/last-commit/ivoronin/ec2ssh/main)
![GitHub Workflow Status (with event)](https://img.shields.io/github/actions/workflow/status/ivoronin/ec2ssh/goreleaser.yml)
![GitHub top language](https://img.shields.io/github/languages/top/ivoronin/ec2ssh)

**SSH into EC2 instances the easy way.** No more juggling instance IDs, IP lookups, and key management.

![](demo/demo.webp)

That's it. You're connected.

## Features

### 🔍 Smart Instance Discovery

Connect using whatever identifier you have - no more digging through the AWS console:

| Identifier | Example |
|------------|---------|
| Instance ID | `ec2ssh i-0123456789abcdef0` |
| Name tag | `ec2ssh my-app-server` |
| Private IP | `ec2ssh 10.0.1.50` |
| Public IP | `ec2ssh 54.123.45.67` |
| IPv6 | `ec2ssh 2600:1f18:...` |
| Private DNS | `ec2ssh ip-10-0-1-50.ec2.internal` |

ec2ssh auto-detects the identifier type, or specify explicitly with `--destination-type`.

### 🔑 Ephemeral SSH Keys via EC2 Instance Connect

No more managing SSH keys. For each session ec2ssh:

1. Generates a fresh ed25519 keypair
2. Pushes the public key to the instance via EC2 Instance Connect API
3. Connects using the private key
4. Cleans up keys after disconnection

**Security**: Keys are valid for only 60 seconds on the instance. No SSH keys are stored permanently - maximum security, zero maintenance.

### 🚇 Private Instance Access via EICE

Reach instances in private subnets without bastion hosts or VPNs:

```bash
ec2ssh --use-eice my-private-server
```

ec2ssh handles WebSocket tunnel setup and AWS Signature V4 signing automatically. EICE endpoint is auto-detected based on instance VPC/subnet.

### 📋 Additional Features

- **Full SSH/SFTP/SCP Compatibility** - All options pass through: `-L`, `-D`, `-J`, `-o`, and more
- **Quick Instance Listing** - See all instances with `ec2list` or `--list`

## Installation

### Using Go

```bash
go install github.com/ivoronin/ec2ssh@latest
```

### From Releases

Download the latest binary for your platform from [GitHub Releases](https://github.com/ivoronin/ec2ssh/releases).

### Symlinks

Create symlinks to use `ec2sftp`, `ec2scp`, and `ec2list` as standalone commands:

```bash
ln -s ec2ssh ec2sftp
ln -s ec2ssh ec2scp
ln -s ec2ssh ec2list
```

The binary auto-detects its intent based on the name it was invoked with.

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

## License

GPL-3.0 - See [LICENSE](LICENSE) for details.
