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

// Session represents an SSH connection session to an EC2 instance.
type Session struct {
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
	sshArgs         []string
	commandWithArgs []string
	logger          *log.Logger
}

// NewSession creates a new SSH session for connecting to an EC2 instance.
func NewSession(
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
) (*Session, error) {
	instance, err := client.GetInstance(dstType, destination)
	if err != nil {
		return nil, fmt.Errorf("unable to get instance: %w", err)
	}

	session := &Session{
		client:          client,
		instance:        instance,
		login:           login,
		port:            port,
		useEICE:         useEICE,
		eiceID:          eiceID,
		noSendKeys:      noSendKeys,
		sshArgs:         sshArgs,
		commandWithArgs: commandWithArgs,
		logger:          logger,
	}

	err = session.setupDestinationAddr(addrType)
	if err != nil {
		return nil, err
	}

	err = session.setupSSHKeys(identityFile, tmpDir)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Session) setupDestinationAddr(addrType ec2.AddrType) error {
	var err error

	if s.useEICE {
		s.destinationAddr = *s.instance.InstanceId
		s.proxyCommand = fmt.Sprintf("%s --wscat", os.Args[0])
	} else {
		s.destinationAddr, err = ec2.GetInstanceAddr(s.instance, addrType)
	}

	return err
}

func (s *Session) setupSSHKeys(identityFile, tmpDir string) error {
	var err error

	if identityFile == "" {
		s.privateKeyPath, s.publicKey, err = GenerateKeypair(tmpDir)
	} else {
		s.privateKeyPath = identityFile
		s.publicKey, err = GetPublicKey(identityFile)
	}

	return err
}

func (s *Session) buildSSHArgs() []string {
	var sshArgs []string

	appendIfSet := func(option, value string) {
		if value != "" {
			sshArgs = append(sshArgs, fmt.Sprintf(option, value))
		}
	}

	appendIfSet("-oProxyCommand=%s", s.proxyCommand)
	appendIfSet("-l%s", s.login)
	appendIfSet("-p%s", s.port)
	appendIfSet("-i%s", s.privateKeyPath)

	sshArgs = append(sshArgs, fmt.Sprintf("-oHostKeyAlias=%s", *s.instance.InstanceId))
	sshArgs = append(sshArgs, s.sshArgs...)
	sshArgs = append(sshArgs, s.destinationAddr)

	if len(s.commandWithArgs) > 0 {
		sshArgs = append(sshArgs, "--")
		sshArgs = append(sshArgs, s.commandWithArgs...)
	}

	return sshArgs
}

// Run executes the SSH connection.
func (s *Session) Run() error {
	var err error

	if !s.noSendKeys {
		err = s.client.SendSSHPublicKey(s.instance, s.login, s.publicKey)
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

	if s.useEICE {
		tunnelURI, err := s.client.CreateEICETunnelURI(s.instance, s.port, s.eiceID)
		if err != nil {
			return fmt.Errorf("unable to setup EICE tunnel: %w", err)
		}

		cmd.Env = append(cmd.Env, fmt.Sprintf("EC2SSH_TUNNEL_URI=%s", tunnelURI))
	}

	s.logger.Printf("running ssh with args: %v", sshArgs)

	exitCode := 0

	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) { // Don't print error message if ssh exits with non-zero exit code
			return err
		}

		exitCode = exitError.ExitCode()
	}

	s.logger.Printf("ssh exited with code %d", exitCode)

	return nil
}
