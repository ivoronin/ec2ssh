package main

import (
	"fmt"
	"os"
	"os/exec"
)

func HandleError(err error) {
	fmt.Fprintf(os.Stderr, "ec2ssh: error: %v\n", err)
	os.Exit(1)
}

func HandleWarning(msg string) {
	fmt.Fprintf(os.Stderr, "ec2ssh: warning: %s\n", msg)
}

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: ec2ssh [--ssh-public-key path] [--use-public-ip] [--region region]\n")
	fmt.Fprintf(os.Stderr, "        [--destination-type <id|private_ip|public_ip|private_dns|name_tag>]\n")
	fmt.Fprintf(os.Stderr, "        [--no-send-keys]\n")
	fmt.Fprintf(os.Stderr, "        [-l login_user] [other ssh flags] destination [command [argument ...]]\n")
	os.Exit(1)
}

func main() {
	opts, sshArgs := ParseArgs()

	EC2Init(opts)

	dstType := opts.dstType
	if dstType == DstTypeUnknown {
		dstType = GuessDestinationType(sshArgs.Destination())
	}

	var instanceID string
	switch dstType {
	case DstTypeID:
		instanceID = sshArgs.Destination()
	case DstTypePrivateIP:
		instanceID = GetInstanceIDByFilter("private-ip-address", sshArgs.Destination())
	case DstTypePublicIP:
		instanceID = GetInstanceIDByFilter("ip-address", sshArgs.Destination())
	case DstTypePrivateDNSName:
		instanceID = GetInstanceIDByFilter("private-dns-name", sshArgs.Destination()+".*")
	case DstTypeNameTag:
		instanceID = GetInstanceIDByFilter("tag:Name", sshArgs.Destination())
	}

	if dstType != DstTypePrivateIP && dstType != DstTypePublicIP {
		ip := GetInstanceIPByID(instanceID, opts.usePublicIP)
		sshArgs.SetDestination(ip)
	} else if opts.usePublicIP {
		HandleWarning("the option '--use-public-ip' is ignored since an IP address has been provided")
	}

	if !opts.noSendKeys {
		SendSSHPublicKey(instanceID, opts.loginUser, opts.sshPublicKeyPath)
	}

	cmd := exec.Command("ssh", sshArgs.Args()...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		HandleError(err)
	}
}
