package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/ivoronin/ec2ssh/internal/app"
	"github.com/ivoronin/ec2ssh/internal/cli/argsieve"
	"github.com/ivoronin/ec2ssh/internal/intent"
	"github.com/ivoronin/ec2ssh/internal/tunnel"
)

// Note: Help is handled by intent.IntentHelp, not by ErrHelp from option parsing.

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
	// Resolve intent from binary name and first argument
	resolvedIntent, args := intent.Resolve(os.Args[0], os.Args[1:])

	var err error

	switch resolvedIntent {
	case intent.IntentTunnel:
		tunnelURI := os.Getenv("EC2SSH_TUNNEL_URI")
		if tunnelURI == "" {
			fatalError(errors.New("EC2SSH_TUNNEL_URI environment variable not set"))
		}

		err = tunnel.Run(tunnelURI)
	case intent.IntentHelp:
		usage(nil)

		return
	case intent.IntentList:
		err = app.RunList(args)
	case intent.IntentSSH:
		err = app.RunSSH(args)
	case intent.IntentSFTP:
		err = app.RunSFTP(args)
	default:
		fatalError(fmt.Errorf("unhandled intent: %v", resolvedIntent))
	}

	if err != nil {
		switch {
		case errors.Is(err, argsieve.ErrSift), errors.Is(err, app.ErrUsage):
			usage(err)
		default:
			fatalError(err)
		}
	}
}
