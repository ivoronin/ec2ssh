package app

import (
	"fmt"
	"os/user"
	"reflect"
	"slices"

	"github.com/ivoronin/ec2ssh/internal/cli"
	"github.com/ivoronin/ec2ssh/internal/ec2"
)

// Options holds the parsed configuration for an ec2ssh session.
type Options struct {
	DstType         ec2.DstType
	AddrType        ec2.AddrType
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
	DoList          bool
	ListColumns     string
}

func parseType[T ~int](value string, mapping map[string]T) (T, error) { //nolint: ireturn // #37
	if value, ok := mapping[value]; ok {
		return value, nil
	}

	return 0, fmt.Errorf("%w: unknown type %s", cli.ErrParse, value)
}

func parseDstType(value string) (ec2.DstType, error) {
	return parseType(value, map[string]ec2.DstType{
		"":            ec2.DstTypeAuto,
		"id":          ec2.DstTypeID,
		"private_ip":  ec2.DstTypePrivateIP,
		"public_ip":   ec2.DstTypePublicIP,
		"ipv6":        ec2.DstTypeIPv6,
		"private_dns": ec2.DstTypePrivateDNSName,
		"name_tag":    ec2.DstTypeNameTag,
	})
}

func parseAddrType(value string) (ec2.AddrType, error) {
	return parseType(value, map[string]ec2.AddrType{
		"":        ec2.AddrTypeAuto,
		"private": ec2.AddrTypePrivate,
		"public":  ec2.AddrTypePublic,
		"ipv6":    ec2.AddrTypeIPv6,
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
		"--list":             &options.DoList,
		"--list-columns":     &options.ListColumns,
	} {
		// check if argument is not present or option is already set
		value, ok := argsOptions[option]
		if !ok || !reflect.ValueOf(variablePtr).Elem().IsZero() {
			continue
		}

		switch variable := variablePtr.(type) {
		case *string:
			*variable = value
		case *bool:
			*variable = true
		case *ec2.DstType:
			*variable, err = parseDstType(value)
		case *ec2.AddrType:
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

func validateListOptions(options map[string]string) error {
	allowedListOptions := [...]string{
		"--list-columns",
		"--region",
		"--profile",
		"--list",
	}
	listOnlyOptions := [...]string{
		"--list-columns",
	}

	if _, ok := options["--list"]; !ok {
		for option := range options {
			if slices.Contains(listOnlyOptions[:], option) {
				return fmt.Errorf("%w: option %s is only allowed when using --list", cli.ErrParse, listOnlyOptions[0])
			}
		}

		return nil
	}

	for option := range options {
		if !slices.Contains(allowedListOptions[:], option) {
			return fmt.Errorf("%w: option %s is not allowed when using --list", cli.ErrParse, option)
		}
	}

	return nil
}

// NewOptions creates Options from parsed CLI arguments.
func NewOptions(parsedArgs cli.ParsedArgs) (Options, error) {
	login, host, port := cli.ParseSSHDestination(parsedArgs.Destination)

	options := Options{
		Destination:     host,
		Login:           login,
		Port:            port,
		CommandWithArgs: parsedArgs.CommandWithArgs,
		SSHArgs:         parsedArgs.SSHArgs,
	}

	err := validateListOptions(parsedArgs.Options)
	if err != nil {
		return Options{}, err
	}

	err = options.populateFromParsedArgsOptions(parsedArgs.Options)
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
