package app

import (
	"fmt"

	"github.com/ivoronin/argsieve"
	"github.com/ivoronin/ec2ssh/internal/ssh"
)

// SCPSession represents an SCP file transfer to/from an EC2 instance.
type SCPSession struct {
	baseSSHSession

	// SCP-specific fields
	LocalPath string
	IsUpload  bool // true = local→remote, false = remote→local
}

// scpPassthroughWithArg lists SCP short options that take arguments.
// These are passed through to SCP along with their values.
var scpPassthroughWithArg = []string{
	"-c", "-F", "-J", "-l", "-o", "-P", "-S",
}

// NewSCPSession creates an SCPSession from command-line arguments.
func NewSCPSession(args []string) (*SCPSession, error) {
	var session SCPSession

	remaining, positional, err := argsieve.Sift(&session, args, scpPassthroughWithArg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUsage, err)
	}

	// Apply implied flags and validate early
	session.ApplyImpliedFlags()
	if err := session.Validate(); err != nil {
		return nil, err
	}

	// Parse SCP operands (source and target)
	if len(positional) != 2 {
		return nil, fmt.Errorf("%w: scp requires exactly 2 operands", ErrUsage)
	}

	srcLocal := ssh.IsLocalPath(positional[0])
	dstLocal := ssh.IsLocalPath(positional[1])

	switch {
	case srcLocal && dstLocal:
		return nil, fmt.Errorf("%w: no remote operand (use host:path)", ErrUsage)
	case !srcLocal && !dstLocal:
		return nil, fmt.Errorf("%w: multiple remote operands not supported", ErrUsage)
	case !srcLocal:
		// Download: remote → local
		target, err := ssh.NewSCPTarget(positional[0])
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrUsage, err)
		}
		session.Target = target
		session.LocalPath = positional[1]
		session.IsUpload = false
	default:
		// Upload: local → remote
		target, err := ssh.NewSCPTarget(positional[1])
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrUsage, err)
		}
		session.Target = target
		session.LocalPath = positional[0]
		session.IsUpload = true
	}

	session.PassArgs = remaining

	return &session, nil
}

func (s *SCPSession) buildArgs() []string {
	args := s.baseArgs()

	// Skip operands in passthrough mode
	if s.Target == nil {
		return args
	}

	// Order depends on direction (host already set by run())
	if s.IsUpload {
		args = append(args, s.LocalPath, s.Target.String())
	} else {
		args = append(args, s.Target.String(), s.LocalPath)
	}

	return args
}

// Run executes the SCP file transfer.
func (s *SCPSession) Run() error {
	return s.run("scp", s.buildArgs)
}
