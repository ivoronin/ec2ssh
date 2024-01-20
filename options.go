package main

import (
	"fmt"
	"os/user"
	"reflect"
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
		host = stripIPv6Brackets(host)
	} else {
		login, host = parseLoginHost(destination)
	}

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

func parseType[T ~int](value string, mapping map[string]T) (T, error) { //nolint: ireturn // #37
	if value, ok := mapping[value]; ok {
		return value, nil
	}

	return 0, fmt.Errorf("%w: unknown type %s", ErrArgParse, value)
}

func parseDstType(value string) (DstType, error) {
	return parseType(value, map[string]DstType{
		"":            DstTypeAuto,
		"id":          DstTypeID,
		"private_ip":  DstTypePrivateIP,
		"public_ip":   DstTypePublicIP,
		"ipv6":        DstTypeIPv6,
		"private_dns": DstTypePrivateDNSName,
		"name_tag":    DstTypeNameTag,
	})
}

func parseAddrType(value string) (AddrType, error) {
	return parseType(value, map[string]AddrType{
		"":        AddrTypeAuto,
		"private": AddrTypePrivate,
		"public":  AddrTypePublic,
		"ipv6":    AddrTypeIPv6,
	})
}

func (options *Options) populateFromParsedArgsOptions(argsOptions map[string]string) error {
	var err error

	for option, variablePtr := range map[string]any{
		"--region":           &options.Region,
		"--profile":          &options.Profile,
		"--eice-id":          &options.EICEID,
		"--destination":      &options.Destination,
		"-i":                 &options.IdentityFile,
		"-l":                 &options.Login,
		"-p":                 &options.Port,
		"--use-eice":         &options.UseEICE,
		"--no-send-keys":     &options.NoSendKeys,
		"--debug":            &options.Debug,
		"--destination-type": &options.DstType,
		"--address-type":     &options.AddrType,
	} {
		/* check if argument is not present or option is already set */
		value, ok := argsOptions[option]
		if !ok || !reflect.ValueOf(variablePtr).Elem().IsZero() {
			continue
		}

		switch variable := variablePtr.(type) {
		case *string:
			*variable = value
		case *bool:
			*variable = true
		case *DstType:
			*variable, err = parseDstType(value)
		case *AddrType:
			*variable, err = parseAddrType(value)
		default:
			panic(fmt.Sprintf("unknown option type %T", variable))
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func NewOptions(parsedArgs ParsedArgs) (Options, error) {
	login, host, port := parseSSHDestination(parsedArgs.Destination)

	options := Options{
		Destination:     host,
		Login:           login,
		Port:            port,
		CommandWithArgs: parsedArgs.CommandWithArgs,
		SSHArgs:         parsedArgs.SSHArgs,
	}

	err := options.populateFromParsedArgsOptions(parsedArgs.Options)
	if err != nil {
		return Options{}, err
	}

	options.UseEICE = options.UseEICE || options.EICEID != ""

	if options.Login == "" {
		user, err := user.Current()
		if err != nil {
			return Options{}, err
		}

		options.Login = user.Username
	}

	return options, nil
}
