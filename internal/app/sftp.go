package app

import (
	"fmt"

	"github.com/ivoronin/ec2ssh/internal/argsieve"
	"github.com/ivoronin/ec2ssh/internal/ssh"
)

// SFTPSession represents an SFTP connection to an EC2 instance.
type SFTPSession struct {
	baseSSHSession
}

// sftpPassthroughWithArg lists SFTP short options that take arguments.
// These are passed through to SFTP along with their values.
var sftpPassthroughWithArg = []string{
	"-B", "-b", "-c", "-D", "-F", "-J",
	"-l", "-o", "-P", "-R", "-S", "-s", "-X",
}

// NewSFTPSession creates an SFTPSession from command-line arguments.
func NewSFTPSession(args []string) (*SFTPSession, error) {
	var session SFTPSession

	remaining, positional, err := argsieve.Sift(&session, args, sftpPassthroughWithArg)
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
		session.Target, err = ssh.NewSFTPTarget(positional[0])
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrUsage, err)
		}
	}

	session.PassArgs = remaining

	return &session, nil
}

func (s *SFTPSession) buildArgs() []string {
	args := s.baseArgs()

	// Skip destination in passthrough mode
	if s.Target == nil {
		return args
	}

	// Output target string (host already set by run())
	args = append(args, s.Target.String())

	return args
}

// Run executes the SFTP connection.
func (s *SFTPSession) Run() error {
	return s.run("sftp", s.buildArgs)
}
