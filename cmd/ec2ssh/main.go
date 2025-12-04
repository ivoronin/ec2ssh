package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/ivoronin/ec2ssh/internal/app"
	"github.com/ivoronin/ec2ssh/internal/cli/argsieve"
	"github.com/ivoronin/ec2ssh/internal/intent"
	"github.com/ivoronin/ec2ssh/internal/tunnel"
)

func fatalError(err error) {
	fmt.Fprintf(os.Stderr, "ec2ssh: %v\n", err)
	os.Exit(1)
}

func usage(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "ec2ssh: %v\n", err)
	}

	fmt.Fprint(os.Stderr, app.HelpText)
	os.Exit(1)
}

func main() {
	resolvedIntent, args := intent.Resolve(os.Args[0], os.Args[1:])

	var err error

	switch resolvedIntent {
	case intent.IntentHelp:
		usage(nil)
	case intent.IntentTunnel:
		tunnelURI := os.Getenv("EC2SSH_TUNNEL_URI")
		if tunnelURI == "" {
			fatalError(errors.New("EC2SSH_TUNNEL_URI environment variable not set"))
		}

		err = tunnel.Run(tunnelURI)
	case intent.IntentList:
		err = app.RunList(args)
	case intent.IntentSSH:
		var session *app.SSHSession
		if session, err = app.NewSSHSession(args); err == nil {
			err = session.Run()
		}
	case intent.IntentSFTP:
		var session *app.SFTPSession
		if session, err = app.NewSFTPSession(args); err == nil {
			err = session.Run()
		}
	case intent.IntentSCP:
		var session *app.SCPSession
		if session, err = app.NewSCPSession(args); err == nil {
			err = session.Run()
		}
	default:
		fatalError(fmt.Errorf("unhandled intent: %v", resolvedIntent))
	}

	if err != nil {
		// Handle subprocess exit codes - propagate them as our exit code
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		} else if errors.Is(err, argsieve.ErrSift) || errors.Is(err, app.ErrUsage) {
			usage(err)
		} else {
			fatalError(err)
		}
	}
}
