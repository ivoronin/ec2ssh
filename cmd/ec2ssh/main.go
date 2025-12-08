package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/ivoronin/ec2ssh/internal/app"
	"github.com/ivoronin/ec2ssh/internal/intent"
)

// version is set at build time via ldflags.
var version = "dev"

// Runner encapsulates the CLI execution logic for testing.
type Runner struct {
	Args   []string  // Command-line arguments (os.Args)
	Stderr io.Writer // Error output writer
}

// DefaultRunner creates a Runner with production defaults.
func DefaultRunner() *Runner {
	return &Runner{
		Args:   os.Args,
		Stderr: os.Stderr,
	}
}

// Run executes the CLI and returns an exit code.
// This method is testable - it doesn't call os.Exit().
func (r *Runner) Run() int {
	resolvedIntent, args := intent.Resolve(r.Args[0], r.Args[1:])

	var err error

	switch resolvedIntent {
	case intent.IntentHelp:
		return r.usage(nil)
	case intent.IntentVersion:
		fmt.Println(version)
		return 0
	case intent.IntentSSH:
		var session *app.SSHSession
		if session, err = app.NewSSHSession(args); err == nil {
			err = session.Run()
		}
	case intent.IntentSCP:
		var session *app.SCPSession
		if session, err = app.NewSCPSession(args); err == nil {
			err = session.Run()
		}
	case intent.IntentSFTP:
		var session *app.SFTPSession
		if session, err = app.NewSFTPSession(args); err == nil {
			err = session.Run()
		}
	case intent.IntentEICETunnel:
		var session *app.EICETunnelSession
		if session, err = app.NewEICETunnelSession(args); err == nil {
			err = session.Run()
		}
	case intent.IntentSSMSession:
		var session *app.SSMSession
		if session, err = app.NewSSMSession(args); err == nil {
			err = session.Run()
		}
	case intent.IntentSSMTunnel:
		var session *app.SSMTunnelSession
		if session, err = app.NewSSMTunnelSession(args); err == nil {
			err = session.Run()
		}
	case intent.IntentList:
		err = app.RunList(args)
	default:
		return r.fatalError(fmt.Errorf("unhandled intent: %v", resolvedIntent))
	}

	if err != nil {
		// Handle subprocess exit codes - propagate them as our exit code
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		} else if errors.Is(err, app.ErrUsage) {
			return r.usage(err)
		} else {
			return r.fatalError(err)
		}
	}

	return 0
}

func (r *Runner) fatalError(err error) int {
	fmt.Fprintf(r.Stderr, "ec2ssh: %v\n", err)
	return 1
}

func (r *Runner) usage(err error) int {
	if err != nil {
		fmt.Fprintf(r.Stderr, "ec2ssh: %v\n", err)
	}
	fmt.Fprint(r.Stderr, HelpText)
	return 1
}

func main() {
	os.Exit(DefaultRunner().Run())
}
