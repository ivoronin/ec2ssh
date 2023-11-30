package main

import (
	"fmt"
	"os"
	"os/exec"
)

func handleError(err error) {
	fmt.Fprintf(os.Stderr, "ec2ssh: error: %v\n", err)
	os.Exit(1)
}

func handleWarning(msg string) {
	fmt.Fprintf(os.Stderr, "ec2ssh: warning: %s\n", msg)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: ec2ssh [--ssh-public-key path] [--use-public-ip] [--region region]\n")
	fmt.Fprintf(os.Stderr, "        [--destination-type <id|private_ip|public_ip|private_dns|name_tag>]\n")
	fmt.Fprintf(os.Stderr, "        [-l login_user] [other ssh flags] destination [command [argument ...]]\n")
	os.Exit(1)
}

func main() {
	opts, sshArgs := parseArgs()

	ec2init(opts)

	dstType := opts.dstType
	if dstType == DstTypeUnknown {
		dstType = guessDestinationType(sshArgs.Destination())
	}

	var instanceID string
	switch dstType {
	case DstTypeID:
		instanceID = sshArgs.Destination()
	case DstTypePrivateIP:
		instanceID = getInstanceIDByFilter("private-ip-address", sshArgs.Destination())
	case DstTypePublicIP:
		instanceID = getInstanceIDByFilter("ip-address", sshArgs.Destination())
	case DstTypePrivateDNSName:
		instanceID = getInstanceIDByFilter("private-dns-name", sshArgs.Destination()+".*")
	case DstTypeNameTag:
		instanceID = getInstanceIDByFilter("tag:Name", sshArgs.Destination())
	}

	if dstType != DstTypePrivateIP && dstType != DstTypePublicIP {
		ip := getInstanceIPByID(instanceID, opts.usePublicIP)
		sshArgs.SetDestination(ip)
	} else if opts.usePublicIP {
		handleWarning("the option '--use-public-ip' is ignored since an IP address has been provided")
	}

	sendSSHPublicKey(instanceID, opts.loginUser, opts.sshPublicKeyPath)

	cmd := exec.Command("ssh", sshArgs.Args()...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		handleError(err)
	}
}
