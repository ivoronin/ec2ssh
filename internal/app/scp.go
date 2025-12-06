package app

import (
	"fmt"

	"github.com/ivoronin/ec2ssh/internal/cli"
	"github.com/ivoronin/ec2ssh/internal/cli/argsieve"
)

// SCPSession represents an SCP file transfer to/from an EC2 instance.
type SCPSession struct {
	baseSSHSession

	// SCP-specific fields
	LocalPath  string
	RemotePath string
	IsUpload   bool // true = local→remote, false = remote→local

	// Parsing-only fields (not used at runtime, but must be exported for argsieve)
	PortFlag string `short:"P"` // SCP uses uppercase -P for port
}

// scpPassthroughWithArg lists SCP short options that take arguments.
// These are passed through to SCP along with their values.
var scpPassthroughWithArg = []string{
	"-c", "-F", "-J", "-l", "-o", "-S",
}

// NewSCPSession creates an SCPSession from command-line arguments.
func NewSCPSession(args []string) (*SCPSession, error) {
	var session SCPSession

	sieve := argsieve.New(&session, scpPassthroughWithArg)

	remaining, positional, err := sieve.Sift(args)
	if err != nil {
		return nil, err
	}

	// Parse SCP operands (source and target)
	parsed, err := cli.ParseSCPOperands(positional)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUsage, err)
	}

	session.Destination = parsed.Host
	session.RemotePath = parsed.RemotePath
	session.LocalPath = parsed.LocalPath
	session.IsUpload = parsed.IsUpload
	session.Login = parsed.Login

	// Apply port from flag if provided
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

func (s *SCPSession) buildArgs() []string {
	args := s.baseArgs()
	args = appendOptArg(args, "-P%s", s.Port) // SCP uses uppercase -P for port

	// Build remote spec: [login@]host:path
	var remoteSpec string
	if s.Login != "" {
		remoteSpec = s.Login + "@"
	}
	remoteSpec += s.destinationAddr + ":" + s.RemotePath

	// Order depends on direction
	if s.IsUpload {
		args = append(args, s.LocalPath, remoteSpec)
	} else {
		args = append(args, remoteSpec, s.LocalPath)
	}

	return args
}

// Run executes the SCP file transfer.
func (s *SCPSession) Run() error {
	return s.run("scp", s.buildArgs)
}
