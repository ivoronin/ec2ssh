package main

import (
	"fmt"
	"os"

	"github.com/ivoronin/ec2ssh/wscat"
)

func FatalError(err error) {
	fmt.Fprintf(os.Stderr, "ec2ssh: %v\n", err)
	os.Exit(1)
}

func Usage(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "ec2ssh: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "Usage: ec2ssh [--public-key path] [--use-public-ip] [--region region] [--profile profile]\n")
	fmt.Fprintf(os.Stderr, "        [--destination-type <id|private_ip|public_ip|private_dns|name_tag>]\n")
	fmt.Fprintf(os.Stderr, "        [--no-generate-keys] [--use-eice] [--eice-id id]\n")
	fmt.Fprintf(os.Stderr, "        [other ssh flags] destination [command [argument ...]]\n")
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
		ec2ssh()
	}
}
