package app

import (
	"fmt"

	"github.com/ivoronin/argsieve"
	"github.com/ivoronin/ec2ssh/internal/ssh"
)

// SSHSession represents an SSH connection to an EC2 instance.
type SSHSession struct {
	baseSSHSession

	// SSH-specific fields
	CommandWithArgs []string

	// Login captures the -l flag, passed through to SSH.
	// Also used in EC2IC fallback chain: Target.Login() → -l flag → OS user.
	Login string `short:"l"`
}

// sshPassthroughWithArg lists SSH short options that take arguments.
// These are passed through to SSH along with their values.
var sshPassthroughWithArg = []string{
	"-B", "-b", "-c", "-D", "-E", "-e", "-F", "-I",
	"-J", "-L", "-m", "-O", "-o", "-p", "-P", "-R", "-S", "-W", "-w",
}

// NewSSHSession creates an SSHSession from command-line arguments.
func NewSSHSession(args []string) (*SSHSession, error) {
	var session SSHSession

	remaining, positional, err := argsieve.Sift(&session, args, sshPassthroughWithArg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUsage, err)
	}

	// Apply implied flags and validate early
	session.ApplyImpliedFlags()
	if err := session.Validate(); err != nil {
		return nil, err
	}

	// Parse target from first positional
	if len(positional) > 0 {
		session.Target, err = ssh.NewSSHTarget(positional[0])
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrUsage, err)
		}
	}

	session.PassArgs = remaining

	if len(positional) > 1 {
		session.CommandWithArgs = positional[1:]
	}

	// Copy -l flag to baseSession for EC2IC fallback chain
	session.loginFlag = session.Login

	return &session, nil
}

func (s *SSHSession) buildArgs() []string {
	args := s.baseArgs()

	// Pass -l only if provided via flag (not embedded in target)
	args = appendOptArg(args, "-l%s", s.Login)

	// Skip destination in passthrough mode
	if s.Target == nil {
		return args
	}

	// Output target string (host already set by run())
	args = append(args, s.Target.String())

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
