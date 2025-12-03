package app

// HelpText contains the usage documentation for ec2ssh.
const HelpText = `Usage: ec2ssh [intent] [options] [destination] [command ...]
       ec2sftp [options] [destination[:path]]
       ec2scp [options] source target
       ec2list [options]

Connect to EC2 instances via SSH/SFTP/SCP or list available instances.

Intents:
  --ssh       Connect to EC2 instance via SSH (default)
  --sftp      Transfer files to/from EC2 instance via SFTP
  --scp       Copy files to/from EC2 instance via SCP
  --list      List EC2 instances in the region
  --help, -h  Show this help message

  The intent can also be determined by the binary name:
    ec2list   equivalent to ec2ssh --list
    ec2sftp   equivalent to ec2ssh --sftp
    ec2scp    equivalent to ec2ssh --scp

SSH Examples:
  Connect to an instance using the instance ID:
     $ ec2ssh -l ec2-user i-0123456789abcdef0

  Connect to an instance using a name tag with the public IP address:
     $ ec2ssh -p 2222 --address-type public ec2-user@app01

  Connect to an instance using its private DNS name via an EICE tunnel:
     $ ec2ssh --use-eice ip-10-0-0-1

  Use any SSH options and arguments as usual:
     $ ec2ssh --use-eice -L 8888:127.0.0.1:8888 -N -i ~/.ssh/id_rsa_alt app01

SFTP Examples:
  Transfer files to an instance:
     $ ec2sftp user@app01:/remote/path
     $ ec2ssh --sftp -P 2222 user@i-0123456789abcdef0

  Use EICE tunnel for SFTP:
     $ ec2sftp --use-eice user@app01

SCP Examples:
  Copy file from EC2 instance to local:
     $ ec2scp user@i-0123456789abcdef0:/remote/file.txt ./local/
     $ ec2scp user@app01:/etc/config.yaml .

  Copy file from local to EC2 instance:
     $ ec2scp ./local/file.txt user@app01:/remote/path/
     $ ec2scp -r ./local/dir user@i-0123456789abcdef0:/remote/

  Use EICE tunnel for SCP:
     $ ec2scp --use-eice ./file.txt user@app01:/path/

List Examples:
  List all instances in the default region:
     $ ec2ssh --list
     $ ec2list

  List instances with custom columns:
     $ ec2list --list-columns ID,NAME,STATE,TYPE

SSH Options:
  -l <login>
     Login name for SSH connection.

  -p <port>
     Port number for SSH connection.

  -i <identity_file>
     Identity file (private key) for SSH authentication.
     If not specified, ephemeral keys are generated and sent via EC2 Instance Connect.

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
     First matched instance will be used for connection.

  --address-type <private|public|ipv6>
     Specify the address type for connecting to the instance.
     Defaults to use the first available address from the list: private, public, ipv6.

  --no-send-keys
     Do not send SSH keys to the instance using EC2 Instance Connect.

  --debug
     Enable debug logging.

  Additional SSH arguments
     Any unrecognized options are passed through to SSH.

  destination
     Specify the destination for connection. Can be one of: instance ID,
     private, public or IPv6 IP address, private DNS name, or name tag.

SFTP Options:
  -P <port>
     Port number for SFTP connection. Note: uppercase -P (unlike SSH's -p).

  -i <identity_file>
     Identity file (private key) for SFTP authentication.

  --region, --profile, --use-eice, --eice-id, --destination-type, --address-type
     Same as SSH options above.

  --no-send-keys
     Do not send SSH keys to the instance using EC2 Instance Connect.

  --debug
     Enable debug logging.

  Additional SFTP arguments
     Any unrecognized options are passed through to SFTP.

  destination
     Specify destination as [user@]host[:path] or sftp://[user@]host[:port][/path].

SCP Options:
  -P <port>
     Port number for SCP connection. Note: uppercase -P (like SFTP).

  -r
     Recursively copy directories.

  -i <identity_file>
     Identity file (private key) for SCP authentication.

  --region, --profile, --use-eice, --eice-id, --destination-type, --address-type
     Same as SSH options above.

  --no-send-keys
     Do not send SSH keys to the instance using EC2 Instance Connect.

  --debug
     Enable debug logging.

  Additional SCP arguments
     Any unrecognized options are passed through to SCP.

  source target
     Exactly one of source or target must be remote ([user@]host:path).
     The other must be a local path. Multiple remote sources are not supported.

List Options:
  --region <string>
     Use the specified AWS region.

  --profile <string>
     Use the specified AWS profile.

  --list-columns <columns>
     Specify columns to display in the list output.
     Defaults to ID,NAME,STATE,PRIVATE-IP,PUBLIC-IP
     Available columns: ID,NAME,STATE,TYPE,AZ,PRIVATE-IP,PUBLIC-IP,IPV6,PRIVATE-DNS,PUBLIC-DNS

  --debug
     Enable debug logging.
`
