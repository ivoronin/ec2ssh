package main

import (
	"fmt"
	"os"
	"os/user"
	"strings"
)

type Opts struct {
	loginUser        string
	sshPublicKeyPath string
	usePublicIP      bool
	destinationType  DestinationType
}

type SSHArgs struct {
	args   []string
	dstIdx int
}

func (a SSHArgs) Destination() string {
	return a.args[a.dstIdx]
}

func (a SSHArgs) SetDestination(destination string) {
	a.args[a.dstIdx] = destination
}

func (a SSHArgs) Args() []string {
	return a.args
}

func parseArgs() (Opts, SSHArgs) {
	args := os.Args[1:]
	if len(args) < 1 {
		usage()
	}

	usr, err := user.Current()
	if err != nil {
		handleError(err)
	}

	/* default values */
	opts := Opts{
		loginUser:        "ec2-user",
		sshPublicKeyPath: usr.HomeDir + "/.ssh/id_rsa.pub",
		usePublicIP:      false,
		destinationType:  Unknown,
	}

	sshArgs := SSHArgs{
		args:   make([]string, 0, len(args)),
		dstIdx: -1,
	}

	for i := 0; i < len(args); i++ {
		/* ssh doesn't use long keys */
		if strings.HasPrefix(args[i], "--") && len(args[i]) > 2 {
			switch args[i] {
			case "--public-key":
				if i+1 >= len(args) {
					handleError(fmt.Errorf("public key path not provided"))
				}
				opts.sshPublicKeyPath = args[i+1]
				i++
			case "--use-public-ip":
				opts.usePublicIP = true
			case "--destination-type":
				if i+1 >= len(args) {
					handleError(fmt.Errorf("destination type not provided"))
				}
				switch args[i+1] {
				case "id":
					opts.destinationType = ID
				case "private_ip":
					opts.destinationType = PrivateIP
				case "public_ip":
					opts.destinationType = PublicIP
				case "private_dns":
					opts.destinationType = PrivateDNSName
				case "name_tag":
					opts.destinationType = NameTag
				default:
					handleError(fmt.Errorf("unknown destination type: %s", args[i+1]))
				}
				i++
			default:
				handleError(fmt.Errorf("unknown option %s", args[i]))
			}
			continue
		}

		sshArgs.args = append(sshArgs.args, args[i])
		if args[i] == "-l" && i+1 < len(args) {
			opts.loginUser = args[i+1]
			// Skip next argument
			i++
			sshArgs.args = append(sshArgs.args, args[i])
		} else if !strings.HasPrefix(args[i], "-") {
			if sshArgs.dstIdx == -1 {
				sshArgs.dstIdx = len(sshArgs.args) - 1
			}
		}
	}

	if sshArgs.dstIdx == -1 {
		usage()
	}

	return opts, sshArgs
}
