package app

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ivoronin/ec2ssh/internal/ec2"
	"github.com/ivoronin/ec2ssh/internal/ssh"
)


var (
	// ErrHelp indicates help was requested (not a real error).
	ErrHelp = errors.New("help requested")
	// ErrUsage is the parent error for all usage/CLI errors.
	ErrUsage              = errors.New("usage error")
	ErrMissingDestination = fmt.Errorf("%w: missing destination", ErrUsage)
	ErrInvalidListColumns = fmt.Errorf("%w: invalid list columns", ErrUsage)
	ErrUnknownType        = fmt.Errorf("%w: unknown type", ErrUsage)
	ErrInvalidOption      = fmt.Errorf("%w: invalid option", ErrUsage)
)

// Run executes the main ec2ssh workflow with the given command-line arguments.
func Run(args []string) error {
	options, err := NewOptions(args)
	if err != nil {
		return err
	}

	logger := log.New(io.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	if options.Debug {
		logger.SetOutput(os.Stderr)
	}

	client, err := ec2.NewClient(options.Region, options.Profile, logger)
	if err != nil {
		return err
	}

	if options.DoList {
		return runList(client, options)
	}

	if options.Destination == "" {
		return ErrMissingDestination
	}

	tmpDir, err := os.MkdirTemp("", "ec2ssh")
	if err != nil {
		return err
	}

	defer func() { _ = os.RemoveAll(tmpDir) }()

	session, err := ssh.NewSession(client, options.DstType, options.AddrType, options.Destination,
		options.Login, options.Port, options.IdentityFile, options.UseEICE, options.EICEID,
		options.NoSendKeys, options.SSHArgs, options.CommandWithArgs, tmpDir, logger)
	if err != nil {
		return err
	}

	return session.Run()
}

func runList(client *ec2.Client, options Options) error {
	columns, err := parseListColumns(options.ListColumns)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidListColumns, err)
	}

	instances, err := client.ListInstances()
	if err != nil {
		return fmt.Errorf("unable to list instances: %w", err)
	}

	return writeInstanceList(os.Stdout, instances, columns)
}
