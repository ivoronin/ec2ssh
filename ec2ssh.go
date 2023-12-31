package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/ivoronin/ec2ssh/awsutil"
)

var ErrGeneral = errors.New("error")

func GuessAWSDestinationType(dst string) DstType {
	if strings.HasPrefix(dst, "i-") {
		return DstTypeID
	}

	if strings.HasPrefix(dst, "ip-") {
		return DstTypePrivateDNSName
	}

	ip := net.ParseIP(dst)
	if ip != nil {
		if ip.IsPrivate() {
			return DstTypePrivateIP
		}

		return DstTypePublicIP
	}

	return DstTypeNameTag
}

func SetupEICETunnel(sshArgs *SSHArgs, instance *types.Instance, eiceID string) (url string, err error) {
	port := 22

	if portStr := sshArgs.Port(); portStr != "" {
		if port, err = strconv.Atoi(portStr); err != nil {
			return "", fmt.Errorf("%w: ssh port (%s) must be an integer", ErrGeneral, portStr)
		}
	}

	tunnelURI, err := awsutil.CreateInstanceConnectTunnelPresignedURI(*instance, eiceID, port)
	if err != nil {
		return "", err
	}

	sshArgs.SetProxyCommand(fmt.Sprintf("%s --wscat", os.Args[0]))

	return tunnelURI, nil
}

func SetupAndSendSSHKeys(sshArgs *SSHArgs, instance *types.Instance, tmpDir string) (err error) {
	var publicKey string

	privateKeyPath := sshArgs.IdentityFile()
	if privateKeyPath == "" {
		privateKeyPath, publicKey, err = GenerateSSHKeypair(tmpDir)
		if err != nil {
			return err
		}

		sshArgs.SetIdentityFile(privateKeyPath)
	} else {
		publicKey, err = GetSSHPublicKey(privateKeyPath)
		if err != nil {
			return err
		}
	}

	err = awsutil.SendSSHPublicKey(instance, sshArgs.Login(), publicKey)
	if err != nil {
		FatalError(err)
	}

	return nil
}

func RunSSH(sshArgs *SSHArgs, env []string) error {
	cmd := exec.Command("ssh", sshArgs.Args()...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		/* Don't print error message if ssh exits with non-zero exit code */
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return err
		}
	}

	return nil
}

func GetInstance(dstType DstType, destination string) (instance *types.Instance, err error) {
	if dstType == DstTypeUnknown {
		dstType = GuessAWSDestinationType(destination)
	}

	switch dstType {
	case DstTypeID:
		instance, err = awsutil.GetInstanceByID(destination)
	case DstTypePrivateIP:
		instance, err = awsutil.GetInstanceByFilter("private-ip-address", destination)
	case DstTypePublicIP:
		instance, err = awsutil.GetInstanceByFilter("ip-address", destination)
	case DstTypePrivateDNSName:
		instance, err = awsutil.GetInstanceByFilter("private-dns-name", destination+".*")
	case DstTypeNameTag:
		instance, err = awsutil.GetInstanceByFilter("tag:Name", destination)
	case DstTypeUnknown: // silence linter
	}

	return instance, err
}

func SetupDestination(sshArgs *SSHArgs, instance *types.Instance, usePublicIP bool) error {
	sshArgs.SetHostKeyAlias(sshArgs.Destination()) // Save original destination in HostKeyAlias

	if usePublicIP {
		if instance.PublicIpAddress == nil {
			return fmt.Errorf("%w: public IP address not found for instance with ID %s", ErrGeneral, *instance.InstanceId)
		}

		sshArgs.SetDestination(*instance.PublicIpAddress)
	} else {
		if instance.PrivateIpAddress == nil {
			return fmt.Errorf("%w: private IP address not found for instance with ID %s", ErrGeneral, *instance.InstanceId)
		}
		sshArgs.SetDestination(*instance.PrivateIpAddress)
	}

	return nil
}

func ec2ssh() {
	opts, sshArgs, err := ParseArgs(os.Args[1:])
	if err != nil {
		Usage(err)
	}

	if sshArgs.destination == "" {
		FatalError(fmt.Errorf("%w: no destination specified", ErrGeneral))
	}

	if opts.useEICE && opts.usePublicIP {
		FatalError(fmt.Errorf("%w: EC2 Instance Connect Endpoint (EICE) cannot be used with a public IP address", ErrGeneral))
	}

	if err = awsutil.Init(opts.region, opts.profile); err != nil {
		FatalError(err)
	}

	tmpDir, err := os.MkdirTemp("", "ec2ssh")
	if err != nil {
		FatalError(err)
	}

	defer os.RemoveAll(tmpDir)

	instance, err := GetInstance(opts.dstType, sshArgs.Destination())
	if err != nil {
		FatalError(err)
	}

	err = SetupDestination(sshArgs, instance, opts.usePublicIP)
	if err != nil {
		FatalError(err)
	}

	if !opts.noSendKeys {
		if err = SetupAndSendSSHKeys(sshArgs, instance, tmpDir); err != nil {
			FatalError(err)
		}
	}

	env := os.Environ()

	if opts.useEICE {
		tunnelURI, err := SetupEICETunnel(sshArgs, instance, opts.eiceID)
		if err != nil {
			FatalError(err)
		}

		env = append(env, fmt.Sprintf("EC2SSH_TUNNEL_URI=%s", tunnelURI))
	}

	if err = RunSSH(sshArgs, env); err != nil {
		FatalError(err)
	}
}
