package main

// HelpText contains the usage documentation for ec2ssh.
const HelpText = `Usage: ec2ssh [intent] [options] [user@]destination [command]
       ec2sftp [options] [user@]destination[:path]
       ec2scp [options] source target
       ec2ssm [options] destination
       ec2list [options]

Intents (first argument or inferred from binary name ec2sftp/ec2scp/ec2ssm/ec2list):
  --ssh (default), --sftp, --scp, --ssm, --list, --help, --version

AWS Options:
  --region <region>       AWS region (default: SDK config)
  --profile <profile>     AWS profile (default: SDK config)

Connection Options:
  --use-eice              Use EC2 Instance Connect Endpoint (default: false)
  --use-ssm               Use SSM Session Manager for tunneling (default: false)
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
`
