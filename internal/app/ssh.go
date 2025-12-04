package app

import (
	"github.com/ivoronin/ec2ssh/internal/cli"
	"github.com/ivoronin/ec2ssh/internal/cli/argsieve"
)

// SSHSession represents an SSH connection to an EC2 instance.
type SSHSession struct {
	baseSession

	// SSH-specific fields
	CommandWithArgs []string

	// Parsing-only fields (not used at runtime, but must be exported for argsieve)
	LoginFlag string `short:"l"` // SSH uses -l for login
	PortFlag  string `short:"p"` // SSH uses lowercase -p for port
}

// sshPassthroughWithArg lists SSH short options that take arguments.
// These are passed through to SSH along with their values.
var sshPassthroughWithArg = []string{
	"-B", "-b", "-c", "-D", "-E", "-e", "-F", "-I",
	"-J", "-L", "-m", "-O", "-o", "-P", "-R", "-S", "-W", "-w",
}

// NewSSHSession creates an SSHSession from command-line arguments.
func NewSSHSession(args []string) (*SSHSession, error) {
	var session SSHSession

	sieve := argsieve.New(&session, sshPassthroughWithArg)

	remaining, positional, err := sieve.Sift(args)
	if err != nil {
		return nil, err
	}

	// Parse destination from first positional (may contain user@host:port)
	if len(positional) > 0 {
		login, host, port := cli.ParseSSHDestination(positional[0])
		session.Destination = host
		session.Login = login
		session.Port = port
	}

	// Flags override destination-parsed values
	if session.LoginFlag != "" {
		session.Login = session.LoginFlag
	}
	if session.PortFlag != "" {
		session.Port = session.PortFlag
	}

	session.PassArgs = remaining

	if len(positional) > 1 {
		session.CommandWithArgs = positional[1:]
	}

	// Parse type strings to enums
	if err := session.ParseTypes(); err != nil {
		return nil, err
	}

	// Apply common defaults (EICE ID implies UseEICE, default login)
	if err := session.ApplyDefaults(); err != nil {
		return nil, err
	}

	if session.Destination == "" {
		return nil, ErrMissingDestination
	}

	return &session, nil
}

func (s *SSHSession) buildArgs() []string {
	args := s.baseArgs()
	args = appendOptArg(args, "-l%s", s.Login)
	args = appendOptArg(args, "-p%s", s.Port)
	args = append(args, s.destinationAddr)

	if len(s.CommandWithArgs) > 0 {
		args = append(args, "--")
		args = append(args, s.CommandWithArgs...)
	}

	return args
}

// Run executes the SSH connection.
func (s *SSHSession) Run() error {
	return s.run("ssh", s.buildArgs)
}
