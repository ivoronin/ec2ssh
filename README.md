# ec2ssh

![GitHub release (with filter)](https://img.shields.io/github/v/release/ivoronin/ec2ssh)
[![Go Report Card](https://goreportcard.com/badge/github.com/ivoronin/ec2ssh)](https://goreportcard.com/report/github.com/ivoronin/ec2ssh)
![GitHub last commit (branch)](https://img.shields.io/github/last-commit/ivoronin/ec2ssh/main)
![GitHub Workflow Status (with event)](https://img.shields.io/github/actions/workflow/status/ivoronin/ec2ssh/build-and-release.yml)
![GitHub top language](https://img.shields.io/github/languages/top/ivoronin/ec2ssh)

**The Swiss Army knife for EC2 instance access.** SSH, SCP, SFTP, and SSM - one tool, zero configuration.

> Stop juggling instance IDs, managing SSH keys, and maintaining bastion hosts.
> ec2ssh gives you secure access to any EC2 instance in seconds.

![](demo/demo.webp)

## Why ec2ssh?

| Challenge | Traditional Approach | ec2ssh |
|-----------|---------------------|--------|
| Finding instances | Copy instance ID from AWS console | Use name tag, IP, or DNS directly |
| SSH key management | Distribute, rotate, revoke keys across teams | Ephemeral keys - auto-generated, 60s lifetime |
| Private instance access | Maintain bastion hosts and VPNs | Built-in EICE and SSM tunneling |
| Multiple protocols | Juggle separate tools for SSH/SCP/SFTP | One binary, multiple modes |
| Dependencies | Python, Ruby, or Node runtimes | Single Go binary, zero dependencies |

## At a Glance

| Capability | Command |
|------------|---------|
| SSH to instance | `ec2ssh my-server` |
| SCP file transfer | `ec2scp ./file.txt ec2-user@my-server:/tmp/` |
| SFTP session | `ec2sftp my-server` |
| SSM shell (no SSH) | `ec2ssm my-server` |
| SSM run command | `ec2ssm my-server whoami` |
| List all instances | `ec2list` or `ec2ssh --list` |
| Private instance via EICE | `ec2ssh --use-eice my-private-server` |
| Private instance via SSM | `ec2ssh --use-ssm my-private-server` |
| Port forwarding | `ec2ssh -L 8080:localhost:80 my-server` |

## Features

### Connection Methods

ec2ssh supports four ways to reach your instances:

| Method | Command | Use Case | Requirements |
|--------|---------|----------|--------------|
| **Direct SSH** | `ec2ssh` | Public instances or within VPC | Network connectivity to instance |
| **EICE Tunnel** | `ec2ssh --use-eice` | Private instances via EC2 Instance Connect Endpoint | EICE endpoint in VPC |
| **SSM Tunnel** | `ec2ssh --use-ssm` | SSH over Systems Manager tunnel | SSM Agent on instance |
| **SSM Shell** | `ec2ssm` | Direct shell via SSM (no SSH) | SSM Agent on instance |
| **SSM RunCommand** | `ec2ssm host cmd` | Execute command via SSM API | SSM Agent on instance |

```bash
# Direct SSH (public instance or same VPC)
ec2ssh my-public-server

# SSH via EC2 Instance Connect Endpoint (private subnet)
ec2ssh --use-eice my-private-server

# SSH via SSM tunnel (no inbound ports required)
ec2ssh --use-ssm my-private-server

# Direct SSM shell session - no SSH at all
ec2ssm my-instance

# Execute command via SSM RunCommand API
ec2ssm my-instance whoami
ec2ssm my-instance -- ls -la /tmp
```

### Smart Instance Discovery

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

### Ephemeral SSH Keys

No more managing SSH keys. For each session ec2ssh:

1. Generates a fresh ed25519 keypair
2. Pushes the public key to the instance via EC2 Instance Connect API
3. Connects using the private key
4. Cleans up after disconnection

**Security**: Keys are valid for only 60 seconds on the instance. No SSH keys stored permanently - maximum security, zero maintenance.

### Full SSH Compatibility

All standard SSH/SCP/SFTP options pass through unchanged:

```bash
# Port forwarding (access RDS through EC2)
ec2ssh -L 3306:my-rds.cluster.us-east-1.rds.amazonaws.com:3306 my-server

# Remote port forwarding
ec2ssh -R 8080:localhost:3000 my-server

# Jump host
ec2ssh -J bastion my-private-server

# Custom SSH options
ec2ssh -o StrictHostKeyChecking=no my-server
```

## Installation

### Homebrew (macOS)

```bash
brew install ivoronin/ivoronin/ec2ssh
```

### Binary Download

Download the latest binary for your platform from [GitHub Releases](https://github.com/ivoronin/ec2ssh/releases).

### Symlinks

The binary auto-detects its mode based on the name it was invoked with. Create symlinks to use standalone commands:

```bash
ln -s ec2ssh ec2scp
ln -s ec2ssh ec2sftp
ln -s ec2ssh ec2ssm
ln -s ec2ssh ec2list
```

## Configuration

### AWS Credentials

ec2ssh uses the standard AWS SDK credential chain:

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

**Basic usage** (SSH/SCP/SFTP):
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

**EICE tunneling** (`--use-eice`) - add:
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

**SSM access** (`--use-ssm` or `ec2ssm`) - add:
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

## Usage Reference

<details>
<summary><strong>Full command reference (click to expand)</strong></summary>

```
Usage: ec2ssh [options] [user@]destination [command [args...]]
       ec2scp [options] source target
       ec2sftp [options] [user@]destination[:path]
       ec2ssm [options] destination [command [args...]]
       ec2list [options]

Intents (first argument or binary name ec2ssh/ec2scp/ec2sftp/ec2ssm/ec2list):
  --ssh (default), --scp, --sftp, --ssm, --list

AWS Options:
  --region <region>       AWS region (default: SDK config)
  --profile <profile>     AWS profile (default: SDK config)

Connection Options:
  --use-eice              Use EC2 Instance Connect Endpoint (default: false)
  --use-ssm               Use SSM Session Manager for tunneling (default: false)
  --eice-id <id>          EICE ID (implies --use-eice, default: autodetect)
  --destination-type <t>  How to interpret destination (default: auto)
                          Values: id|private_ip|public_ip|ipv6|private_dns|name_tag
  --address-type <type>   Address for connection (default: auto)
                          Values: private|public|ipv6
  --no-send-keys          Skip EC2 Instance Connect key push (default: false)

List Options:
  --list-columns <cols>   Columns to display
                          Default: ID,NAME,STATE,PRIVATE-IP,PUBLIC-IP
                          Available: ID,NAME,STATE,TYPE,AZ,PRIVATE-IP,
                                     PUBLIC-IP,IPV6,PRIVATE-DNS,PUBLIC-DNS

SSM Command Options:
  --timeout <duration>    Timeout for command completion (default: 60s)

Other:
  --help, --version       Show help or version
  --debug                 Enable debug logging (default: false)

Examples:
  ec2ssh ec2-user@i-0123456789abcdef0
  ec2ssh --use-eice -L 8080:localhost:80 ubuntu@my-web-server
  ec2ssh --use-ssm admin@i-0123456789abcdef0
  ec2scp -r --region us-west-2 ./logs admin@10.0.1.5:/backup/
  ec2sftp -P 2222 user@app01:/var/log
  ec2ssm my-bastion-host
  ec2ssm i-0123456789abcdef0 whoami
  ec2ssm --timeout 5m i-xxx -- ./long-running-script.sh
  ec2list --profile prod --list-columns ID,NAME,STATE

All standard ssh/scp/sftp options are passed through to the underlying command.
```

</details>

## Compared to Alternatives

ec2ssh combines capabilities that typically require multiple tools:

- **vs. AWS CLI `ssh`**: Adds SCP, SFTP, SSM shell, instance listing, smart discovery by name/IP
- **vs. aws-gate**: Single Go binary (no Python), EICE support, full SSH option passthrough
- **vs. aws-ssm-tools**: Native Go, EICE support, ephemeral keys built-in

**One tool. All scenarios. Zero dependencies.**

## Acknowledgments

- [ssm-session-client](https://github.com/mmmorris1975/ssm-session-client) - Pure Go implementation of SSM session protocol

## License

GPL-3.0 - See [LICENSE](LICENSE) for details.
