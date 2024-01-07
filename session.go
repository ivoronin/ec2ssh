package main

import (
	"fmt"
	"os/user"
	"strings"
)

type DstType int

const (
	DstTypeAuto DstType = iota
	DstTypeID
	DstTypePrivateIP
	DstTypePublicIP
	DstTypeIPv6
	DstTypePrivateDNSName
	DstTypeNameTag
)

type AddrType int

const (
	AddrTypeAuto AddrType = iota
	AddrTypePrivate
	AddrTypePublic
	AddrTypeIPv6
)

type Session struct {
	dstType         DstType
	addrType        AddrType
	region          string
	profile         string
	eiceID          string
	useEICE         bool
	noSendKeys      bool
	sshArgs         []string
	commandWithArgs []string
	destination     string
	port            string
	login           string
	identityFile    string
	proxyCommand    string
	hostKeyAlias    string
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

func parseDstType(value string) (DstType, error) {
	dstTypeNames := map[string]DstType{
		"":            DstTypeAuto,
		"id":          DstTypeID,
		"private_ip":  DstTypePrivateIP,
		"public_ip":   DstTypePublicIP,
		"ipv6":        DstTypeIPv6,
		"private_dns": DstTypePrivateDNSName,
		"name_tag":    DstTypeNameTag,
	}

	if dstType, ok := dstTypeNames[value]; ok {
		return dstType, nil
	}

	return 0, fmt.Errorf("%w: unknown destination type %s", ErrArgParse, value)
}

func parseAddrType(value string) (AddrType, error) {
	addrTypeNames := map[string]AddrType{
		"":        AddrTypeAuto,
		"private": AddrTypePrivate,
		"public":  AddrTypePublic,
		"ipv6":    AddrTypeIPv6,
	}

	if addrType, ok := addrTypeNames[value]; ok {
		return addrType, nil
	}

	return 0, fmt.Errorf("%w: unknown address type %s", ErrArgParse, value)
}

func NewSession(parsedArgs *ParsedArgs) (*Session, error) {
	var err error

	if parsedArgs.Destination == "" {
		return nil, fmt.Errorf("%w: missing destination", ErrArgParse)
	}

	login, host, port := parseSSHDestination(parsedArgs.Destination)

	session := &Session{
		destination:     host,
		login:           login,
		port:            port,
		dstType:         DstTypeAuto,
		addrType:        AddrTypeAuto,
		commandWithArgs: parsedArgs.CommandWithArgs,
		sshArgs:         parsedArgs.SSHArgs,
	}

	optionToVariableMap := map[string]*string{
		"--region":      &session.region,
		"--profile":     &session.profile,
		"--eice-id":     &session.eiceID,
		"--destination": &session.destination,
		"-i":            &session.identityFile,
		"-l":            &session.login,
		"-p":            &session.port,
	}

	for option, variable := range optionToVariableMap {
		if *variable == "" { /* do not override destination, login and port */
			*variable = parsedArgs.Options[option]
		}
	}

	_, session.useEICE = parsedArgs.Options["--use-eice"]
	_, session.noSendKeys = parsedArgs.Options["--no-send-keys"]

	session.dstType, err = parseDstType(parsedArgs.Options["--destination-type"])
	if err != nil {
		return nil, err
	}

	session.addrType, err = parseAddrType(parsedArgs.Options["--address-type"])
	if err != nil {
		return nil, err
	}

	session.useEICE = session.useEICE || session.eiceID != "" /* eiceID implies useEICE */

	if session.login == "" { /* default login to current user */
		user, err := user.Current()
		if err != nil {
			return nil, err
		}

		session.login = user.Username
	}

	return session, nil
}

func (s *Session) Destination() string {
	return s.destination
}

func (s *Session) SetDestination(dst string) {
	s.destination = dst
}

func (s *Session) IdentityFile() string {
	return s.identityFile
}

func (s *Session) SetIdentityFile(identityFile string) {
	s.identityFile = identityFile
}

func (s *Session) Port() string {
	return s.port
}

func (s *Session) Login() string {
	return s.login
}

func (s *Session) SetProxyCommand(proxyCommand string) {
	s.proxyCommand = proxyCommand
}

func (s *Session) SetHostKeyAlias(alias string) {
	s.hostKeyAlias = alias
}

func (s *Session) BuildSSHArgs() []string {
	sshArgs := make([]string, 0)

	if s.proxyCommand != "" {
		sshArgs = append(sshArgs, fmt.Sprintf("-oProxyCommand=%s", s.proxyCommand))
	}

	if s.hostKeyAlias != "" {
		sshArgs = append(sshArgs, fmt.Sprintf("-oHostKeyAlias=%s", s.hostKeyAlias))
	}

	if s.login != "" {
		sshArgs = append(sshArgs, fmt.Sprintf("-l%s", s.login))
	}

	if s.port != "" {
		sshArgs = append(sshArgs, fmt.Sprintf("-p%s", s.port))
	}

	if s.identityFile != "" {
		sshArgs = append(sshArgs, fmt.Sprintf("-i%s", s.identityFile))
	}

	sshArgs = append(sshArgs, s.sshArgs...)
	sshArgs = append(sshArgs, s.destination)

	if len(s.commandWithArgs) > 0 {
		sshArgs = append(sshArgs, "--")
		sshArgs = append(sshArgs, s.commandWithArgs...)
	}

	return sshArgs
}
