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

	addr := net.ParseIP(dst)
	if addr == nil {
		return DstTypeNameTag
	}

	if addr.To4() != nil {
		if addr.IsPrivate() {
			return DstTypePrivateIP
		}

		return DstTypePublicIP
	}

	return DstTypeIPv6
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
		return err
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
	case DstTypeIPv6:
		instance, err = awsutil.GetInstanceByFilter("ipv6-address", destination)
	case DstTypePrivateDNSName:
		instance, err = awsutil.GetInstanceByFilter("private-dns-name", destination+".*")
	case DstTypeNameTag:
		instance, err = awsutil.GetInstanceByFilter("tag:Name", destination)
	case DstTypeUnknown: // silence linter
	}

	return instance, err
}

func SetupDestination(sshArgs *SSHArgs, instance *types.Instance, addrType AddrType) error {
	sshArgs.SetHostKeyAlias(*instance.InstanceId)

	switch addrType {
	case AddrTypePrivate:
		if instance.PrivateIpAddress == nil {
			return fmt.Errorf("%w: private IP address not found for instance with ID %s", ErrGeneral, *instance.InstanceId)
		}

		sshArgs.SetDestination(*instance.PrivateIpAddress)
	case AddrTypePublic:
		if instance.PublicIpAddress == nil {
			return fmt.Errorf("%w: public IP address not found for instance with ID %s", ErrGeneral, *instance.InstanceId)
		}

		sshArgs.SetDestination(*instance.PublicIpAddress)
	case AddrTypeIPv6:
		if instance.Ipv6Address == nil {
			return fmt.Errorf("%w: IPv6 address not found for instance with ID %s", ErrGeneral, *instance.InstanceId)
		}

		sshArgs.SetDestination(*instance.Ipv6Address)
	}

	return nil
}

func ec2ssh(opts *Opts, sshArgs *SSHArgs) (err error) {
	if sshArgs.destination == "" {
		return fmt.Errorf("%w: no destination specified", ErrGeneral)
	}

	if opts.useEICE && opts.addrType != AddrTypePrivate {
		return fmt.Errorf("%w: EC2 Instance Connect Endpoint (EICE) can be used only with private addresses", ErrGeneral)
	}

	if err = awsutil.Init(opts.region, opts.profile); err != nil {
		return fmt.Errorf("unable to initialize AWS SDK: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "ec2ssh")
	if err != nil {
		return fmt.Errorf("unable to create temporary directory: %w", err)
	}

	defer os.RemoveAll(tmpDir)

	instance, err := GetInstance(opts.dstType, sshArgs.Destination())
	if err != nil {
		return fmt.Errorf("unable to get instance: %w", err)
	}

	err = SetupDestination(sshArgs, instance, opts.addrType)
	if err != nil {
		return fmt.Errorf("unable to setup destination: %w", err)
	}

	if !opts.noSendKeys {
		if err = SetupAndSendSSHKeys(sshArgs, instance, tmpDir); err != nil {
			return fmt.Errorf("unable to setup and send SSH keys: %w", err)
		}
	}

	env := os.Environ()

	if opts.useEICE {
		tunnelURI, err := SetupEICETunnel(sshArgs, instance, opts.eiceID)
		if err != nil {
			return fmt.Errorf("unable to setup EICE tunnel: %w", err)
		}

		env = append(env, fmt.Sprintf("EC2SSH_TUNNEL_URI=%s", tunnelURI))
	}

	if err = RunSSH(sshArgs, env); err != nil {
		return fmt.Errorf("unable to run ssh: %w", err)
	}

	return nil
}
