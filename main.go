package main

import (
	"fmt"
	"os"
	"os/exec"

	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
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
	fmt.Fprintf(os.Stderr, "        [--no-send-keys] [--use-eice] [--eice-id id]\n")
	fmt.Fprintf(os.Stderr, "        [-l login_user] [other ssh flags] destination [command [argument ...]]\n")
	os.Exit(1)
}

func main() {
	opts, sshArgs := ParseArgs()

	AWSInit(opts)

	dstType := opts.dstType
	if dstType == DstTypeUnknown {
		dstType = GuessDestinationType(sshArgs.Destination())
	}

	var instance *ec2Types.Instance
	switch dstType {
	case DstTypeID:
		instance = GetInstanceById(sshArgs.Destination())
	case DstTypePrivateIP:
		instance = GetInstanceByFilter("private-ip-address", sshArgs.Destination())
		if opts.usePublicIP {
			HandleWarning("the option '--use-public-ip' is ignored since an IP address has been provided")
		}
	case DstTypePublicIP:
		instance = GetInstanceByFilter("ip-address", sshArgs.Destination())
		if opts.usePublicIP {
			HandleWarning("the option '--use-public-ip' is ignored since an IP address has been provided")
		}
	case DstTypePrivateDNSName:
		instance = GetInstanceByFilter("private-dns-name", sshArgs.Destination()+".*")
	case DstTypeNameTag:
		instance = GetInstanceByFilter("tag:Name", sshArgs.Destination())
	}

	var ip string
	if opts.usePublicIP {
		if instance.PublicIpAddress == nil {
			HandleError(fmt.Errorf("public IP address not found for instance with ID %s", *instance.InstanceId))
		}
		ip = *instance.PublicIpAddress
	} else {
		if instance.PrivateIpAddress == nil {
			HandleError(fmt.Errorf("private IP address not found for instance with ID %s", *instance.InstanceId))
		}
		ip = *instance.PrivateIpAddress
	}
	sshArgs.SetDestination(ip)

	if !opts.noSendKeys {
		SendSSHPublicKey(*instance.InstanceId, opts.loginUser, opts.sshPublicKeyPath)
	}

	if opts.useEICE {
		var eice *ec2Types.Ec2InstanceConnectEndpoint
		if opts.eiceId != "" {
			eice = GetInstanceConnectEndpointByID(opts.eiceId)
		} else {
			eice = GetInstanceConnectEndpointByVpc(*instance.VpcId, *instance.SubnetId)
		}
		fmt.Printf("using EICE: %s\n", *eice.DnsName)
	}

	cmd := exec.Command("ssh", sshArgs.Args()...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		HandleError(err)
	}
}
