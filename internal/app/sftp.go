package app

import (
	"fmt"

	"github.com/ivoronin/ec2ssh/internal/cli"
	"github.com/ivoronin/ec2ssh/internal/argsieve"
)

// SFTPSession represents an SFTP connection to an EC2 instance.
type SFTPSession struct {
	baseSSHSession

	// SFTP-specific fields
	RemotePath string

	// Parsing-only fields (not used at runtime, but must be exported for argsieve)
	PortFlag string `short:"P"` // SFTP uses uppercase -P for port
}

// sftpPassthroughWithArg lists SFTP short options that take arguments.
// These are passed through to SFTP along with their values.
var sftpPassthroughWithArg = []string{
	"-B", "-b", "-c", "-D", "-F", "-J",
	"-l", "-o", "-R", "-S", "-s", "-X",
}

// NewSFTPSession creates an SFTPSession from command-line arguments.
func NewSFTPSession(args []string) (*SFTPSession, error) {
	var session SFTPSession

	sieve := argsieve.New(&session, sftpPassthroughWithArg)

	remaining, positional, err := sieve.Sift(args)
	if err != nil {
		return nil, err
	}

	// Parse destination from first positional (may contain user@host:path)
	if len(positional) > 0 {
		login, host, port, path := cli.ParseSFTPDestination(positional[0])
		session.Destination = host
		session.RemotePath = path
		session.Login = login
		session.Port = port
	}

	// Flag overrides destination-parsed value
	if session.PortFlag != "" {
		session.Port = session.PortFlag
	}

	session.PassArgs = remaining

	// Parse type strings to enums
	if err := session.ParseTypes(); err != nil {
		return nil, err
	}

	// Apply common defaults (EICE ID implies UseEICE, default login)
	if err := session.ApplyDefaults(); err != nil {
		return nil, err
	}

	if session.Destination == "" {
		return nil, fmt.Errorf("%w: missing destination", ErrUsage)
	}

	return &session, nil
}

func (s *SFTPSession) buildArgs() []string {
	args := s.baseArgs()
	args = appendOptArg(args, "-P%s", s.Port) // SFTP uses uppercase -P for port

	// Build destination: [login@]host[:path]
	var destination string
	if s.Login != "" {
		destination = s.Login + "@"
	}
	destination += s.destinationAddr
	if s.RemotePath != "" {
		destination += ":" + s.RemotePath
	}
	args = append(args, destination)

	return args
}

// Run executes the SFTP connection.
func (s *SFTPSession) Run() error {
	return s.run("sftp", s.buildArgs)
}
