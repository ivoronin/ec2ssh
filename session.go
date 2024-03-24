package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/ivoronin/ec2ssh/awsutil"
)

type Session struct {
	options         Options
	instance        types.Instance
	destinationAddr string
	privateKeyPath  string
	publicKey       string
	proxyCommand    string
}

func (s *Session) buildSSHArgs() []string {
	var sshArgs []string

	appendIfSet := func(option, value string) {
		if value != "" {
			sshArgs = append(sshArgs, fmt.Sprintf(option, value))
		}
	}

	appendIfSet("-oProxyCommand=%s", s.proxyCommand)
	appendIfSet("-l%s", s.options.Login)
	appendIfSet("-p%s", s.options.Port)
	appendIfSet("-i%s", s.privateKeyPath)

	sshArgs = append(sshArgs, fmt.Sprintf("-oHostKeyAlias=%s", *s.instance.InstanceId))
	sshArgs = append(sshArgs, s.options.SSHArgs...)
	sshArgs = append(sshArgs, s.destinationAddr)

	if len(s.options.CommandWithArgs) > 0 {
		sshArgs = append(sshArgs, "--")
		sshArgs = append(sshArgs, s.options.CommandWithArgs...)
	}

	return sshArgs
}

func NewSession(options Options, tmpDir string) (*Session, error) {
	instance, err := GetInstance(options.DstType, options.Destination)
	if err != nil {
		return nil, fmt.Errorf("unable to get instance: %w", err)
	}

	session := &Session{
		options:  options,
		instance: instance,
	}

	err = session.setupDestinationAddr()
	if err != nil {
		return nil, err
	}

	err = session.setupSSHKeys(tmpDir)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Session) setupDestinationAddr() error {
	var err error

	if s.options.UseEICE {
		s.destinationAddr = *s.instance.InstanceId
		s.proxyCommand = fmt.Sprintf("%s --wscat", os.Args[0])
	} else {
		s.destinationAddr, err = GetInstanceAddr(s.instance, s.options.AddrType)
	}

	return err
}

func (s *Session) setupSSHKeys(tmpDir string) error {
	var err error

	if s.options.IdentityFile == "" {
		s.privateKeyPath, s.publicKey, err = GenerateSSHKeypair(tmpDir)
	} else {
		s.privateKeyPath = s.options.IdentityFile
		s.publicKey, err = GetSSHPublicKey(s.options.IdentityFile)
	}

	return err
}

func (s *Session) Run() error {
	var err error

	if !s.options.NoSendKeys {
		err = awsutil.SendSSHPublicKey(s.instance, s.options.Login, s.publicKey)
		if err != nil {
			return fmt.Errorf("unable to send SSH public key: %w", err)
		}
	}

	sshArgs := s.buildSSHArgs()
	cmd := exec.Command("ssh", sshArgs...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if s.options.UseEICE {
		tunnelURI, err := awsutil.CreateEICETunnelURI(s.instance, s.options.Port, s.options.EICEID)
		if err != nil {
			return fmt.Errorf("unable to setup EICE tunnel: %w", err)
		}

		cmd.Env = append(cmd.Env, fmt.Sprintf("EC2SSH_TUNNEL_URI=%s", tunnelURI))
	}

	DebugLogger.Printf("running ssh with args: %v", sshArgs)

	exitCode := 0

	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) { /* Don't print error message if ssh exits with non-zero exit code */
			return err
		}

		exitCode = exitError.ExitCode()
	}

	DebugLogger.Printf("ssh exited with code %d", exitCode)

	return nil
}
