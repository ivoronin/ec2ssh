package app

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ivoronin/ec2ssh/internal/cli"
	"github.com/ivoronin/ec2ssh/internal/ec2"
	"github.com/ivoronin/ec2ssh/internal/ssh"
)

// Run executes the main ec2ssh workflow with the given command-line arguments.
func Run(args []string) error {
	parsedArgs, err := cli.ParseArgs(args)
	if err != nil {
		return err
	}

	options, err := NewOptions(parsedArgs)
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
		return fmt.Errorf("%w: missing destination", cli.ErrParse)
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
		return fmt.Errorf("%w: %v", cli.ErrParse, err)
	}

	instances, err := client.ListInstances()
	if err != nil {
		return fmt.Errorf("unable to list instances: %w", err)
	}

	return writeInstanceList(os.Stdout, instances, columns)
}
