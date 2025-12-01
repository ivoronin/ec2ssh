package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/ivoronin/ec2ssh/internal/app"
	"github.com/ivoronin/ec2ssh/internal/cli/argsieve"
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
	tunnelURI := os.Getenv("EC2SSH_TUNNEL_URI")
	if len(os.Args) == 2 && os.Args[1] == "--wscat" && tunnelURI != "" {
		// Run in tunnel mode
		err := tunnel.Run(tunnelURI)
		if err != nil {
			fatalError(err)
		}
	} else {
		// Run in normal mode
		if err := app.Run(os.Args[1:]); err != nil {
			switch {
			case errors.Is(err, app.ErrHelp):
				usage(nil)
			case errors.Is(err, argsieve.ErrSift), errors.Is(err, app.ErrUsage):
				usage(err)
			default:
				fatalError(err)
			}
		}
	}
}
