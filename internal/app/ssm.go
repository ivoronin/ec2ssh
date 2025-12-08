package app

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ivoronin/ec2ssh/internal/awsclient"
	"github.com/ivoronin/ec2ssh/internal/ssh"
	"github.com/ivoronin/ec2ssh/internal/argsieve"
	"github.com/ivoronin/ec2ssh/internal/ec2client"
	"github.com/mmmorris1975/ssm-session-client/ssmclient"
)

// SSMSession represents an SSM Session Manager shell connection to an EC2 instance.
type SSMSession struct {
	// CLI Configuration
	Region  string            `long:"region"`
	Profile string            `long:"profile"`
	DstType ec2client.DstType `long:"destination-type"`
	Debug   bool              `long:"debug"`

	// Parsed values
	Destination string

	// Runtime
	logger *log.Logger
}

// NewSSMSession creates an SSMSession from command-line arguments.
func NewSSMSession(args []string) (*SSMSession, error) {
	var session SSMSession

	positional, err := argsieve.Parse(&session, args)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUsage, err)
	}

	// Parse destination from first positional
	if len(positional) > 0 {
		target, err := ssh.NewSSHTarget(positional[0])
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrUsage, err)
		}
		session.Destination = target.Host()
	}

	// Reject extra positional arguments
	if len(positional) > 1 {
		return nil, fmt.Errorf("%w: unexpected argument %s", ErrUsage, positional[1])
	}

	if session.Destination == "" {
		return nil, fmt.Errorf("%w: missing destination", ErrUsage)
	}

	return &session, nil
}

// Run starts the SSM session.
func (s *SSMSession) Run() error {
	// Initialize logger
	s.logger = log.New(io.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	if s.Debug {
		s.logger.SetOutput(os.Stderr)
	}

	// Load AWS config
	cfg, err := awsclient.LoadConfig(s.Region, s.Profile, s.logger)
	if err != nil {
		return err
	}

	// Create EC2 client to resolve instance
	client, err := newEC2Client(cfg, s.logger)
	if err != nil {
		return err
	}

	// Get instance
	instance, err := client.GetInstance(s.DstType, s.Destination)
	if err != nil {
		return err
	}

	if instance.InstanceId == nil {
		panic("ec2ssh: AWS returned instance without InstanceId - this should never happen")
	}

	s.logger.Printf("starting SSM session to instance %s", *instance.InstanceId)

	// Start SSM shell session using the AWS config
	return ssmclient.ShellSession(cfg, *instance.InstanceId)
}
