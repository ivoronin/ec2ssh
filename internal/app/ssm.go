package app

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/ivoronin/argsieve"
	"github.com/ivoronin/ec2ssh/internal/awsclient"
	"github.com/ivoronin/ec2ssh/internal/ec2client"
	"github.com/ivoronin/ec2ssh/internal/ssh"
	"github.com/ivoronin/ec2ssh/internal/ssmcommand"
	"github.com/mmmorris1975/ssm-session-client/ssmclient"
)

// Duration wraps time.Duration to implement encoding.TextUnmarshaler for CLI parsing.
type Duration time.Duration

// UnmarshalText implements encoding.TextUnmarshaler for CLI flag parsing.
func (d *Duration) UnmarshalText(text []byte) error {
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}
	*d = Duration(parsed)
	return nil
}

// SSMSession represents an SSM Session Manager shell connection to an EC2 instance.
// When CommandWithArgs is empty, starts an interactive shell.
// When CommandWithArgs is set, executes command via SSM RunCommand API.
type SSMSession struct {
	// CLI Configuration
	Region         string             `long:"region"`
	Profile        string             `long:"profile"`
	DstType        *ec2client.DstType `long:"destination-type"` // nil = auto-detect
	Debug          bool               `long:"debug"`
	CommandTimeout Duration           `long:"timeout"` // Timeout for command execution (default: 60s)

	// Parsed values
	Destination     string
	CommandWithArgs []string // Command to execute (if any)

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

	// Extra positional arguments become the command
	if len(positional) > 1 {
		session.CommandWithArgs = positional[1:]
	}

	if session.Destination == "" {
		return nil, fmt.Errorf("%w: missing destination", ErrUsage)
	}

	// Set default timeout for command execution
	if session.CommandTimeout == 0 {
		session.CommandTimeout = Duration(60 * time.Second)
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
	instance, err := client.GetInstance(s.Destination, s.DstType)
	if err != nil {
		return err
	}

	if instance.InstanceId == nil {
		panic("ec2ssh: AWS returned instance without InstanceId - this should never happen")
	}

	// Dispatch based on command presence
	if len(s.CommandWithArgs) > 0 {
		s.logger.Printf("running command on instance %s", *instance.InstanceId)

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.CommandTimeout))
		defer cancel()

		stdout, stderr, err := ssmcommand.RunCommand(ctx, cfg, *instance.InstanceId, s.CommandWithArgs)

		// Print output
		_, _ = fmt.Fprint(os.Stdout, stdout)
		_, _ = fmt.Fprint(os.Stderr, stderr)

		return err
	}

	s.logger.Printf("starting SSM session to instance %s", *instance.InstanceId)

	// Start SSM shell session using the AWS config
	return ssmclient.ShellSession(cfg, *instance.InstanceId)
}
