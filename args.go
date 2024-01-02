package main

import (
	"errors"
	"fmt"
	"strings"
)

var ErrArgParse = errors.New("error parsing arguments")

type DstType int

const (
	DstTypeUnknown DstType = iota
	DstTypeID
	DstTypePrivateIP
	DstTypePublicIP
	DstTypeIPv6
	DstTypePrivateDNSName
	DstTypeNameTag
)

type AddrType int

const (
	AddrTypePrivate AddrType = iota
	AddrTypePublic
	AddrTypeIPv6
)

type Opts struct {
	dstType    DstType
	addrType   AddrType
	region     string
	profile    string
	eiceID     string
	useEICE    bool
	noSendKeys bool
}

type SSHArgs struct {
	otherFlags     []string
	commandAndArgs []string
	destination    string
	port           string
	login          string
	identityFile   string
	proxyCommand   string
	hostKeyAlias   string
}

func (a *SSHArgs) Destination() string {
	return a.destination
}

func (a *SSHArgs) IdentityFile() string {
	return a.identityFile
}

func (a *SSHArgs) Port() string {
	return a.port
}

func (a *SSHArgs) Login() string {
	return a.login
}

func (a *SSHArgs) SetDestination(dst string) {
	a.destination = dst
}

func (a *SSHArgs) SetIdentityFile(identityFile string) {
	a.identityFile = identityFile
}

func (a *SSHArgs) SetProxyCommand(proxyCommand string) {
	a.proxyCommand = proxyCommand
}

func (a *SSHArgs) SetHostKeyAlias(alias string) {
	a.hostKeyAlias = alias
}

func (a *SSHArgs) Args() []string {
	args := make([]string, 0)

	if a.proxyCommand != "" {
		args = append(args, fmt.Sprintf("-oProxyCommand=%s", a.proxyCommand))
	}

	if a.hostKeyAlias != "" {
		args = append(args, fmt.Sprintf("-oHostKeyAlias=%s", a.hostKeyAlias))
	}

	if a.login != "" {
		args = append(args, fmt.Sprintf("-l%s", a.login))
	}

	if a.port != "" {
		args = append(args, fmt.Sprintf("-p%s", a.port))
	}

	if a.identityFile != "" {
		args = append(args, fmt.Sprintf("-i%s", a.identityFile))
	}

	args = append(args, a.otherFlags...)
	args = append(args, a.destination)
	args = append(args, a.commandAndArgs...)

	return args
}

func getOptValue(args []string, idx int) (value string, err error) {
	if idx+1 >= len(args) || strings.HasPrefix(args[idx+1], "-") {
		return "", fmt.Errorf("%w: missing argument for %s", ErrArgParse, args[idx])
	}

	return args[idx+1], nil
}

func ParseOpts(args []string) (opts *Opts, leftoverArgs []string, err error) {
	DstTypeNames := map[string]DstType{
		"auto":        DstTypeUnknown,
		"id":          DstTypeID,
		"private_ip":  DstTypePrivateIP,
		"public_ip":   DstTypePublicIP,
		"ipv6":        DstTypeIPv6,
		"private_dns": DstTypePrivateDNSName,
		"name_tag":    DstTypeNameTag,
	}

	AddrTypeNames := map[string]AddrType{
		"private": AddrTypePrivate,
		"public":  AddrTypePublic,
		"ipv6":    AddrTypeIPv6,
	}

	opts = &Opts{dstType: DstTypeUnknown}

	leftoverArgs = []string{}
	/* Pass 1 - parse long options */
	for argIdx := 0; argIdx < len(args); argIdx++ {
		if args[argIdx] == "--" {
			leftoverArgs = append(leftoverArgs, args[argIdx:]...)

			break
		}

		if strings.HasPrefix(args[argIdx], "--") { /* ssh doesn't use long keys, so we do */
			opt, value, includesValue := strings.Cut(args[argIdx], "=")

			done := true

			switch opt { /* parse long options without values */
			case "--no-send-keys":
				opts.noSendKeys = true
			case "--use-eice":
				opts.useEICE = true
			default:
				done = false
			}

			if !done { /* parse long options with values */
				if !includesValue {
					if value, err = getOptValue(args, argIdx); err != nil {
						return nil, nil, err
					}
					argIdx++
				}

				switch opt {
				case "--region":
					opts.region = value
				case "--profile":
					opts.profile = value
				case "--destination-type":
					var ok bool
					if opts.dstType, ok = DstTypeNames[value]; !ok {
						return nil, nil, fmt.Errorf("%w: unknown destination type %s", ErrArgParse, value)
					}
				case "--address-type":
					var ok bool
					if opts.addrType, ok = AddrTypeNames[value]; !ok {
						return nil, nil, fmt.Errorf("%w: unknown connection type %s", ErrArgParse, value)
					}
				case "--eice-id":
					opts.eiceID, opts.useEICE = value, true
				default:
					return nil, nil, fmt.Errorf("%w: unknown option %s", ErrArgParse, opt)
				}
			}

			continue
		}

		leftoverArgs = append(leftoverArgs, args[argIdx])
	}

	return opts, leftoverArgs, nil
}

/* https://github.com/openssh/openssh-portable/blob/V_9_6_P1/ssh.c#L183 */
const sshFlagsWithArguments = "BbcDEeFIiJLlmOoPpRSWw"

func ParseSSHArgs(args []string) (sshArgs *SSHArgs, err error) {
	sshArgs = &SSHArgs{otherFlags: []string{}, commandAndArgs: []string{}}

	for argIdx := 0; argIdx < len(args); argIdx++ {
		if args[argIdx] == "--" {
			sshArgs.commandAndArgs = append(sshArgs.commandAndArgs, args[argIdx:]...)

			break
		}

		if strings.HasPrefix(args[argIdx], "-") && len(args[argIdx]) > 1 {
			flags := args[argIdx][1:]
			/* for each flag in the current argument */
			for flagIdx := 0; flagIdx < len(flags); flagIdx++ {
				flag := flags[flagIdx : flagIdx+1]

				/* current flag doesn't have a value */
				if !strings.Contains(sshFlagsWithArguments, flag) {
					sshArgs.otherFlags = append(sshArgs.otherFlags, "-"+flag)

					continue
				}

				var value string

				/* current flag must have a value */
				if len(flags[flagIdx:]) > 2 {
					/* current flag has an argument attached to it */
					value = flags[flagIdx+1:]
					flagIdx = len(flags) // Stop iterating over current argument
				} else {
					/* current flag must have a value in the next argument */
					if value, err = getOptValue(args, argIdx); err != nil {
						return nil, err
					}
					argIdx++
				}

				switch flag { /* extract login, port and identity values */
				case "l":
					if sshArgs.login == "" {
						sshArgs.login = value
					}
				case "p":
					if sshArgs.port == "" {
						sshArgs.port = value
					}
				case "i":
					if sshArgs.identityFile == "" {
						sshArgs.identityFile = value
					}
				default: /* normalize and save other flags */
					sshArgs.otherFlags = append(sshArgs.otherFlags, "-"+flag, value)
				}
			}

			continue
		}

		if sshArgs.destination == "" {
			login, host, port := parseSSHDestination(args[argIdx])
			sshArgs.destination = host

			if sshArgs.login == "" && login != "" {
				sshArgs.login = login
			}

			if sshArgs.port == "" && port != "" {
				sshArgs.port = port
			}
		} else {
			sshArgs.commandAndArgs = append(sshArgs.commandAndArgs, args[argIdx:]...)

			break
		}
	}

	return sshArgs, nil
}

func parseSSHDestination(destination string) (string, string, string) {
	var login, host, port string

	loginhostport, hasPrefix := strings.CutPrefix(destination, "ssh://")

	if hasPrefix {
		var hostport string

		atIdx := strings.LastIndex(loginhostport, "@")

		if atIdx != -1 {
			before, after := loginhostport[:atIdx], loginhostport[atIdx+1:]
			login = before
			hostport = after
		} else {
			hostport = loginhostport
		}

		colonIdx := strings.LastIndex(hostport, ":")
		/* workaround for IPv6 addresses, e.g. [fec1::1] will give {"[fec1:", "1]"} */
		bracketIdx := strings.LastIndex(hostport, "]")
		if bracketIdx > colonIdx {
			colonIdx = -1
		}

		if colonIdx != -1 {
			before, after := hostport[:colonIdx], hostport[colonIdx+1:]
			host = before
			port = after
		} else {
			host = hostport
		}

		// Strip IPv6 brackets
		if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
			host = host[1 : len(host)-1]
		}
	} else {
		atIdx := strings.LastIndex(destination, "@")
		if atIdx != -1 {
			before, after := destination[:atIdx], destination[atIdx+1:]
			login = before
			host = after
		} else {
			host = destination
		}
	}

	return login, host, port
}

func ParseArgs(args []string) (*Opts, *SSHArgs, error) {
	if len(args) < 1 {
		return nil, nil, fmt.Errorf("%w: no arguments provided", ErrArgParse)
	}

	opts, leftoverArgs, err := ParseOpts(args)
	if err != nil {
		return nil, nil, err
	}

	sshArgs, err := ParseSSHArgs(leftoverArgs)
	if err != nil {
		return nil, nil, err
	}

	return opts, sshArgs, nil
}
