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

type Options struct {
	DstType         DstType
	AddrType        AddrType
	Region          string
	Profile         string
	EICEID          string
	UseEICE         bool
	NoSendKeys      bool
	Debug           bool
	SSHArgs         []string
	CommandWithArgs []string
	Destination     string
	Port            string
	Login           string
	IdentityFile    string
}

func parseSSHDestination(destination string) (string, string, string) {
	var login, host, port string

	if strings.HasPrefix(destination, "ssh://") {
		login, host, port = parseSSHURL(destination)
	} else {
		login, host = parseLoginHost(destination)
	}

	host = stripIPv6Brackets(host)

	return login, host, port
}

func parseSSHURL(url string) (string, string, string) {
	loginhostport := strings.TrimPrefix(url, "ssh://")
	login, hostport := parseLoginHost(loginhostport)
	host, port := parseHostPort(hostport)

	return login, host, port
}

func parseLoginHost(loginhost string) (string, string) {
	atIdx := strings.LastIndex(loginhost, "@")
	if atIdx != -1 {
		return loginhost[:atIdx], loginhost[atIdx+1:]
	}

	return "", loginhost
}

func parseHostPort(hostport string) (string, string) {
	colonIdx := strings.LastIndex(hostport, ":")
	/* handle square brackets around IPv6 addresses */
	if colonIdx != -1 && strings.LastIndex(hostport, "]") < colonIdx {
		return hostport[:colonIdx], hostport[colonIdx+1:]
	}

	return hostport, ""
}

func stripIPv6Brackets(host string) string {
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return host[1 : len(host)-1]
	}

	return host
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

func mapOptionsToVariables(options *Options, parsedArgs ParsedArgs) {
	for option, variable := range map[string]*string{
		"--region":      &options.Region,
		"--profile":     &options.Profile,
		"--eice-id":     &options.EICEID,
		"--destination": &options.Destination,
		"-i":            &options.IdentityFile,
		"-l":            &options.Login,
		"-p":            &options.Port,
	} {
		if value, ok := parsedArgs.Options[option]; ok && *variable == "" {
			*variable = value
		}
	}
}

func setOptionsFromParsedArgs(options *Options, parsedArgs ParsedArgs) error {
	mapOptionsToVariables(options, parsedArgs)

	if _, ok := parsedArgs.Options["--use-eice"]; ok {
		options.UseEICE = true
	}

	if _, ok := parsedArgs.Options["--no-send-keys"]; ok {
		options.NoSendKeys = true
	}

	if _, ok := parsedArgs.Options["--debug"]; ok {
		options.Debug = true
	}

	var err error

	options.DstType, err = parseDstType(parsedArgs.Options["--destination-type"])
	if err != nil {
		return err
	}

	options.AddrType, err = parseAddrType(parsedArgs.Options["--address-type"])
	if err != nil {
		return err
	}

	options.UseEICE = options.UseEICE || options.EICEID != ""

	return nil
}

func setDefaultLogin(options *Options) error {
	user, err := user.Current()
	if err != nil {
		return err
	}

	options.Login = user.Username

	return nil
}

func NewOptions(parsedArgs ParsedArgs) (Options, error) {
	login, host, port := parseSSHDestination(parsedArgs.Destination)

	options := Options{
		Destination:     host,
		Login:           login,
		Port:            port,
		DstType:         DstTypeAuto,
		AddrType:        AddrTypeAuto,
		CommandWithArgs: parsedArgs.CommandWithArgs,
		SSHArgs:         parsedArgs.SSHArgs,
	}

	if err := setOptionsFromParsedArgs(&options, parsedArgs); err != nil {
		return Options{}, err
	}

	if options.Login == "" {
		if err := setDefaultLogin(&options); err != nil {
			return Options{}, err
		}
	}

	return options, nil
}
