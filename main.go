package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ivoronin/ec2ssh/awsutil"
	"github.com/ivoronin/ec2ssh/wscat"
)

func FatalError(err error) {
	fmt.Fprintf(os.Stderr, "ec2ssh: %v\n", err)
	os.Exit(1)
}

var DebugLogger = log.New(io.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

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
`

func Usage(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "ec2ssh: %v\n", err)
	}

	fmt.Fprint(os.Stderr, helpText)
	os.Exit(1)
}

func EnableDebug() {
	DebugLogger.SetOutput(os.Stderr)
}

func Run(args []string) error {
	parsedArgs, err := ParseArgs(args)
	if err != nil {
		return err
	}

	options, err := NewOptions(parsedArgs)
	if err != nil {
		return err
	}

	if options.Debug {
		EnableDebug()
		awsutil.EnableDebug()
	}

	if err := awsutil.Init(options.Region, options.Profile); err != nil {
		return err
	}

	if options.DoList {
		return List(options)
	}

	if options.Destination == "" {
		return fmt.Errorf("%w: missing destination", ErrArgParse)
	}

	tmpDir, err := os.MkdirTemp("", "ec2ssh")
	if err != nil {
		return err
	}

	defer os.RemoveAll(tmpDir)

	session, err := NewSession(options, tmpDir)
	if err != nil {
		return err
	}

	if err = session.Run(); err != nil {
		return err
	}

	return nil
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
		/* Run in normal mode otherwise */
		if err := Run(os.Args[1:]); err != nil {
			switch {
			case errors.Is(err, ErrHelp):
				Usage(nil)
			case errors.Is(err, ErrArgParse):
				Usage(err)
			default:
				FatalError(err)
			}
		}
	}
}
