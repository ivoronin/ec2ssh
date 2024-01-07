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

func SetupEICETunnel(session *Session, instance *types.Instance, eiceID string) (url string, err error) {
	port := 22

	if portStr := session.Port(); portStr != "" {
		if port, err = strconv.Atoi(portStr); err != nil {
			return "", fmt.Errorf("%w: ssh port (%s) must be an integer", ErrGeneral, portStr)
		}
	}

	tunnelURI, err := awsutil.CreateInstanceConnectTunnelPresignedURI(*instance, eiceID, port)
	if err != nil {
		return "", err
	}

	session.SetProxyCommand(fmt.Sprintf("%s --wscat", os.Args[0]))

	return tunnelURI, nil
}

func SetupAndSendSSHKeys(session *Session, instance *types.Instance, tmpDir string) (err error) {
	var publicKey string

	privateKeyPath := session.IdentityFile()
	if privateKeyPath == "" {
		privateKeyPath, publicKey, err = GenerateSSHKeypair(tmpDir)
		if err != nil {
			return err
		}

		session.SetIdentityFile(privateKeyPath)
	} else {
		publicKey, err = GetSSHPublicKey(privateKeyPath)
		if err != nil {
			return err
		}
	}

	err = awsutil.SendSSHPublicKey(instance, session.Login(), publicKey)
	if err != nil {
		return err
	}

	return nil
}

func RunSSH(session *Session, env []string) error {
	cmd := exec.Command("ssh", session.BuildSSHArgs()...)
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
	if dstType == DstTypeAuto {
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
	case DstTypeAuto: // silence linter
	}

	return instance, err
}

func GetInstanceAddr(instance *types.Instance, addrType AddrType, useEICE bool) (string, error) {
	switch addrType {
	case AddrTypeAuto:
		if useEICE {
			return GetInstanceAddr(instance, AddrTypePrivate, useEICE)
		}

		if instance.PrivateIpAddress != nil {
			return GetInstanceAddr(instance, AddrTypePrivate, useEICE)
		}

		if instance.PublicIpAddress != nil {
			return GetInstanceAddr(instance, AddrTypePublic, useEICE)
		}

		if instance.Ipv6Address != nil {
			return GetInstanceAddr(instance, AddrTypeIPv6, useEICE)
		}
	case AddrTypePrivate:
		if instance.PrivateIpAddress == nil {
			return "", fmt.Errorf("%w: private IP address not found for instance ID %s", ErrGeneral, *instance.InstanceId)
		}

		return *instance.PrivateIpAddress, nil
	case AddrTypePublic:
		if instance.PublicIpAddress == nil {
			return "", fmt.Errorf("%w: public IP address not found for instance ID %s", ErrGeneral, *instance.InstanceId)
		}

		return *instance.PublicIpAddress, nil
	case AddrTypeIPv6:
		if instance.Ipv6Address == nil {
			return "", fmt.Errorf("%w: IPv6 address not found for instance ID %s", ErrGeneral, *instance.InstanceId)
		}

		return *instance.Ipv6Address, nil
	}

	/* auto falls through here */
	return "", fmt.Errorf("%w: no IP addresses found for instance with ID %s", ErrGeneral, *instance.InstanceId)
}

func ec2ssh(session *Session) (err error) {
	if session.useEICE && !((session.addrType == AddrTypePrivate) || (session.addrType == AddrTypeAuto)) {
		return fmt.Errorf("%w: EC2 Instance Connect Endpoint (EICE) can be used only with private addresses", ErrGeneral)
	}

	if session.Port() != "" && session.Port() != "22" && session.useEICE {
		return fmt.Errorf("%w: EC2 Instance Connect Endpoint (EICE) can be used only with port 22", ErrGeneral)
	}

	if err = awsutil.Init(session.region, session.profile); err != nil {
		return fmt.Errorf("unable to initialize AWS SDK: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "ec2ssh")
	if err != nil {
		return fmt.Errorf("unable to create temporary directory: %w", err)
	}

	defer os.RemoveAll(tmpDir)

	instance, err := GetInstance(session.dstType, session.Destination())
	if err != nil {
		return fmt.Errorf("unable to get instance: %w", err)
	}

	dstAddr, err := GetInstanceAddr(instance, session.addrType, session.useEICE)
	if err != nil {
		return fmt.Errorf("unable to get destination address: %w", err)
	}

	session.SetHostKeyAlias(*instance.InstanceId)
	session.SetDestination(dstAddr)

	if !session.noSendKeys {
		if err = SetupAndSendSSHKeys(session, instance, tmpDir); err != nil {
			return fmt.Errorf("unable to setup and send SSH keys: %w", err)
		}
	}

	env := os.Environ()

	if session.useEICE {
		tunnelURI, err := SetupEICETunnel(session, instance, session.eiceID)
		if err != nil {
			return fmt.Errorf("unable to setup EICE tunnel: %w", err)
		}

		env = append(env, fmt.Sprintf("EC2SSH_TUNNEL_URI=%s", tunnelURI))
	}

	if err = RunSSH(session, env); err != nil {
		return fmt.Errorf("unable to run ssh: %w", err)
	}

	return nil
}
