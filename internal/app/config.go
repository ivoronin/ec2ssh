package app

import (
	"fmt"
	"os/user"

	"github.com/ivoronin/ec2ssh/internal/cli"
	"github.com/ivoronin/ec2ssh/internal/cli/argsieve"
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

type sieveOptions struct {
	Region       string `long:"region"`
	Profile      string `long:"profile"`
	EICEID       string `long:"eice-id"`
	DstTypeStr   string `long:"destination-type"`
	AddrTypeStr  string `long:"address-type"`
	IdentityFile string `short:"i"`
	Login        string `short:"l"`
	Port         string `short:"p"`
	UseEICE      bool   `long:"use-eice"`
	NoSendKeys   bool   `long:"no-send-keys"`
	Debug        bool   `long:"debug"`
	DoList       bool   `long:"list"`
	ListColumns  string `long:"list-columns"`
	Help         bool   `short:"h" long:"help"`
}

// passthroughWithArg lists SSH short options that take arguments.
// These are passed through to SSH along with their values.
var passthroughWithArg = []string{
	"-B", "-b", "-c", "-D", "-E", "-e", "-F", "-I",
	"-J", "-L", "-m", "-O", "-o", "-P", "-R", "-S", "-W", "-w",
}

func parseType[T ~int](value string, mapping map[string]T) (T, error) { //nolint: ireturn // #37
	if value, ok := mapping[value]; ok {
		return value, nil
	}

	return 0, fmt.Errorf("%w: %s", ErrUnknownType, value)
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

func (options *Options) populateFromSieveOptions(sieved *sieveOptions) error {
	var err error

	options.Region = sieved.Region
	options.Profile = sieved.Profile
	options.EICEID = sieved.EICEID
	options.IdentityFile = sieved.IdentityFile
	options.UseEICE = sieved.UseEICE || sieved.EICEID != ""
	options.NoSendKeys = sieved.NoSendKeys
	options.Debug = sieved.Debug
	options.DoList = sieved.DoList
	options.ListColumns = sieved.ListColumns

	// Flag overrides destination-parsed value
	if sieved.Login != "" {
		options.Login = sieved.Login
	}

	if sieved.Port != "" {
		options.Port = sieved.Port
	}

	// Parse custom types
	options.DstType, err = parseDstType(sieved.DstTypeStr)
	if err != nil {
		return err
	}

	options.AddrType, err = parseAddrType(sieved.AddrTypeStr)
	if err != nil {
		return err
	}

	return nil
}

func validateListOptions(sieved *sieveOptions) error {
	if !sieved.DoList {
		if sieved.ListColumns != "" {
			return fmt.Errorf("%w: --list-columns is only allowed with --list", ErrInvalidOption)
		}

		return nil
	}

	if sieved.IdentityFile != "" || sieved.Login != "" || sieved.Port != "" ||
		sieved.EICEID != "" || sieved.UseEICE || sieved.NoSendKeys ||
		sieved.DstTypeStr != "" || sieved.AddrTypeStr != "" {
		return fmt.Errorf("%w: only --region, --profile, --list-columns are allowed with --list", ErrInvalidOption)
	}

	return nil
}

// NewOptions creates Options from command-line arguments.
func NewOptions(args []string) (Options, error) {
	var sieved sieveOptions

	sieve := argsieve.New(&sieved, passthroughWithArg)

	remaining, positional, err := sieve.Sift(args)
	if err != nil {
		return Options{}, err
	}

	if sieved.Help {
		return Options{}, ErrHelp
	}

	if err := validateListOptions(&sieved); err != nil {
		return Options{}, err
	}

	// Parse destination from first positional (may contain user@host:port)
	var login, host, port string
	if len(positional) > 0 {
		login, host, port = cli.ParseSSHDestination(positional[0])
	}

	options := Options{
		Destination:     host,
		Login:           login,
		Port:            port,
		SSHArgs:         remaining,
		CommandWithArgs: nil,
	}

	if len(positional) > 1 {
		options.CommandWithArgs = positional[1:]
	}

	if err := options.populateFromSieveOptions(&sieved); err != nil {
		return Options{}, err
	}

	if options.Login == "" {
		user, err := user.Current()
		if err != nil {
			return Options{}, err
		}

		options.Login = user.Username
	}

	return options, nil
}
