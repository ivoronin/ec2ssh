package ssh

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/ivoronin/ec2ssh/internal/ec2"
)

// baseSession contains common fields and methods for all session types.
type baseSession struct {
	client          *ec2.Client
	instance        types.Instance
	destinationAddr string
	privateKeyPath  string
	publicKey       string
	proxyCommand    string
	login           string
	port            string
	useEICE         bool
	eiceID          string
	noSendKeys      bool
	passArgs        []string
	logger          *log.Logger
}

func (s *baseSession) setupDestinationAddr(addrType ec2.AddrType) error {
	var err error

	if s.useEICE {
		s.destinationAddr = *s.instance.InstanceId
		s.proxyCommand = fmt.Sprintf("%s --wscat", os.Args[0])
	} else {
		s.destinationAddr, err = ec2.GetInstanceAddr(s.instance, addrType)
	}

	return err
}

func (s *baseSession) setupSSHKeys(identityFile, tmpDir string) error {
	var err error

	if identityFile == "" {
		s.privateKeyPath, s.publicKey, err = GenerateKeypair(tmpDir)
	} else {
		s.privateKeyPath = identityFile
		s.publicKey, err = GetPublicKey(identityFile)
	}

	return err
}

// run executes the session command. Called by embedded types.
func (s *baseSession) run(command string, args []string) error {
	if !s.noSendKeys {
		if err := s.client.SendSSHPublicKey(s.instance, s.login, s.publicKey); err != nil {
			return fmt.Errorf("unable to send SSH public key: %w", err)
		}
	}

	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if s.useEICE {
		tunnelURI, err := s.client.CreateEICETunnelURI(s.instance, s.port, s.eiceID)
		if err != nil {
			return fmt.Errorf("unable to setup EICE tunnel: %w", err)
		}

		cmd.Env = append(cmd.Env, fmt.Sprintf("EC2SSH_TUNNEL_URI=%s", tunnelURI))
	}

	s.logger.Printf("running %s with args: %v", command, args)

	exitCode := 0

	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return err
		}

		exitCode = exitError.ExitCode()
	}

	s.logger.Printf("%s exited with code %d", command, exitCode)

	return nil
}

// SSHSession represents an SSH connection to an EC2 instance.
type SSHSession struct {
	baseSession
	commandWithArgs []string
}

func (s *SSHSession) buildArgs() []string {
	var args []string

	appendIfSet := func(option, value string) {
		if value != "" {
			args = append(args, fmt.Sprintf(option, value))
		}
	}

	appendIfSet("-oProxyCommand=%s", s.proxyCommand)
	appendIfSet("-l%s", s.login)
	appendIfSet("-p%s", s.port)
	appendIfSet("-i%s", s.privateKeyPath)

	args = append(args, fmt.Sprintf("-oHostKeyAlias=%s", *s.instance.InstanceId))
	args = append(args, s.passArgs...)
	args = append(args, s.destinationAddr)

	if len(s.commandWithArgs) > 0 {
		args = append(args, "--")
		args = append(args, s.commandWithArgs...)
	}

	return args
}

// Run executes the SSH connection.
func (s *SSHSession) Run() error {
	return s.run("ssh", s.buildArgs())
}

// NewSSHSession creates a new SSH session for connecting to an EC2 instance.
func NewSSHSession(
	client *ec2.Client,
	dstType ec2.DstType,
	addrType ec2.AddrType,
	destination string,
	login string,
	port string,
	identityFile string,
	useEICE bool,
	eiceID string,
	noSendKeys bool,
	sshArgs []string,
	commandWithArgs []string,
	tmpDir string,
	logger *log.Logger,
) (*SSHSession, error) {
	instance, err := client.GetInstance(dstType, destination)
	if err != nil {
		return nil, fmt.Errorf("unable to get instance: %w", err)
	}

	s := &SSHSession{
		baseSession: baseSession{
			client:     client,
			instance:   instance,
			login:      login,
			port:       port,
			useEICE:    useEICE,
			eiceID:     eiceID,
			noSendKeys: noSendKeys,
			passArgs:   sshArgs,
			logger:     logger,
		},
		commandWithArgs: commandWithArgs,
	}

	if err := s.setupDestinationAddr(addrType); err != nil {
		return nil, err
	}

	if err := s.setupSSHKeys(identityFile, tmpDir); err != nil {
		return nil, err
	}

	return s, nil
}

// SFTPSession represents an SFTP connection to an EC2 instance.
type SFTPSession struct {
	baseSession
	remotePath string
}

func (s *SFTPSession) buildArgs() []string {
	var args []string

	appendIfSet := func(option, value string) {
		if value != "" {
			args = append(args, fmt.Sprintf(option, value))
		}
	}

	appendIfSet("-oProxyCommand=%s", s.proxyCommand)
	appendIfSet("-P%s", s.port) // SFTP uses uppercase -P for port
	appendIfSet("-i%s", s.privateKeyPath)

	args = append(args, fmt.Sprintf("-oHostKeyAlias=%s", *s.instance.InstanceId))
	args = append(args, s.passArgs...)

	// Build destination: login@host[:path]
	destination := s.destinationAddr
	if s.login != "" {
		destination = s.login + "@" + destination
	}
	if s.remotePath != "" {
		destination = destination + ":" + s.remotePath
	}
	args = append(args, destination)

	return args
}

// Run executes the SFTP connection.
func (s *SFTPSession) Run() error {
	return s.run("sftp", s.buildArgs())
}

// NewSFTPSession creates a new SFTP session for file transfer to an EC2 instance.
func NewSFTPSession(
	client *ec2.Client,
	dstType ec2.DstType,
	addrType ec2.AddrType,
	destination string,
	login string,
	port string,
	identityFile string,
	useEICE bool,
	eiceID string,
	noSendKeys bool,
	sftpArgs []string,
	remotePath string,
	tmpDir string,
	logger *log.Logger,
) (*SFTPSession, error) {
	instance, err := client.GetInstance(dstType, destination)
	if err != nil {
		return nil, fmt.Errorf("unable to get instance: %w", err)
	}

	s := &SFTPSession{
		baseSession: baseSession{
			client:     client,
			instance:   instance,
			login:      login,
			port:       port,
			useEICE:    useEICE,
			eiceID:     eiceID,
			noSendKeys: noSendKeys,
			passArgs:   sftpArgs,
			logger:     logger,
		},
		remotePath: remotePath,
	}

	if err := s.setupDestinationAddr(addrType); err != nil {
		return nil, err
	}

	if err := s.setupSSHKeys(identityFile, tmpDir); err != nil {
		return nil, err
	}

	return s, nil
}
