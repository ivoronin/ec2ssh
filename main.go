package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/ivoronin/ec2ssh/wscat"
)

func FatalError(err error) {
	fmt.Fprintf(os.Stderr, "ec2ssh: %v\n", err)
	os.Exit(1)
}

const helpText = `Usage: ec2ssh [ec2ssh options] [ssh arguments] destination [command [argument ...]]

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
     Default is false. Conflicts with --address-type other than 'auto' or 'private'.

  --eice-id <string>
     Specifies the EC2 Instance Connect Endpoint (EICE) ID to use.
     Defaults to autodetection based on the instance's VPC and subnet.
     Automatically implies --use-eice.

  --destination-type <id|private_ip|public_ip|ipv6|private_dns|name_tag>
     Specify the destination type for instance search.
     Defaults to automatically detecting the type based on the destination.

  --address-type <private|public|ipv6>
     Specify the address type for connecting to the instance.
     Defaults to use the first available address from the list: private, public, ipv6.

  --no-send-keys
     Do not send SSH keys to the instance using EC2 Instance Connect.

  ssh arguments
     Specify arguments to pass to SSH.

  destination
     Specify the destination for connection. Can be one of: instance ID,
     private, public or IPv6 IP address, private DNS name, or name tag.
`

func Usage(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "ec2ssh: %v\n", err)
	}

	fmt.Fprint(os.Stderr, helpText)
	os.Exit(1)
}

func main() {
	tunnelURI := os.Getenv("EC2SSH_TUNNEL_URI")
	if len(os.Args) == 2 && os.Args[1] == "--wscat" && tunnelURI != "" {
		/* Run in socat mode */
		err := wscat.Run(tunnelURI)
		if err != nil {
			FatalError(err)
		}
	} else {
		/* Run in ec2ssh mode otherwise */
		parsedArgs, err := ParseArgs(os.Args[1:])
		if err != nil {
			if errors.Is(err, ErrHelp) {
				Usage(nil)
			}
			Usage(err)
		}

		session, err := NewSession(parsedArgs)
		if err != nil {
			FatalError(err)
		}

		err = ec2ssh(session)
		if err != nil {
			FatalError(err)
		}
	}
}
