package app

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/user"

	"github.com/ivoronin/ec2ssh/internal/cli"
	"github.com/ivoronin/ec2ssh/internal/cli/argsieve"
	"github.com/ivoronin/ec2ssh/internal/ec2"
	"github.com/ivoronin/ec2ssh/internal/ssh"
)

// SFTPOptions holds the parsed configuration for an SFTP session.
type SFTPOptions struct {
	// Fields populated by argsieve from flags
	Region       string `long:"region"`
	Profile      string `long:"profile"`
	EICEID       string `long:"eice-id"`
	DstTypeStr   string `long:"destination-type"`
	AddrTypeStr  string `long:"address-type"`
	IdentityFile string `short:"i"`
	Port         string `short:"P"` // SFTP uses uppercase -P for port
	UseEICE      bool   `long:"use-eice"`
	NoSendKeys   bool   `long:"no-send-keys"`
	Debug        bool   `long:"debug"`

	// Fields populated after parsing
	DstType     ec2.DstType
	AddrType    ec2.AddrType
	Destination string
	Login       string
	RemotePath  string
	SFTPArgs    []string
}

// sftpPassthroughWithArg lists SFTP short options that take arguments.
// These are passed through to SFTP along with their values.
var sftpPassthroughWithArg = []string{
	"-B", "-b", "-c", "-D", "-F", "-J",
	"-l", "-o", "-R", "-S", "-s", "-X",
}

// NewSFTPOptions creates SFTPOptions from command-line arguments.
func NewSFTPOptions(args []string) (*SFTPOptions, error) {
	var options SFTPOptions

	sieve := argsieve.New(&options, sftpPassthroughWithArg)

	remaining, positional, err := sieve.Sift(args)
	if err != nil {
		return nil, err
	}

	// Parse destination from first positional (may contain user@host:path)
	if len(positional) > 0 {
		login, host, port, path := cli.ParseSFTPDestination(positional[0])
		options.Destination = host
		options.RemotePath = path
		// Only set from destination if not already set by flags
		if options.Login == "" {
			options.Login = login
		}
		if options.Port == "" {
			options.Port = port
		}
	}

	options.SFTPArgs = remaining

	// Parse type strings to enums
	if err := options.parseTypes(); err != nil {
		return nil, err
	}

	// EICE ID implies UseEICE
	if options.EICEID != "" {
		options.UseEICE = true
	}

	// Default login to current user
	if options.Login == "" {
		u, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("unable to determine current user: %w", err)
		}
		options.Login = u.Username
	}

	return &options, nil
}

func (options *SFTPOptions) parseTypes() error {
	dstTypes := map[string]ec2.DstType{
		"":            ec2.DstTypeAuto,
		"id":          ec2.DstTypeID,
		"private_ip":  ec2.DstTypePrivateIP,
		"public_ip":   ec2.DstTypePublicIP,
		"ipv6":        ec2.DstTypeIPv6,
		"private_dns": ec2.DstTypePrivateDNSName,
		"name_tag":    ec2.DstTypeNameTag,
	}

	dstType, ok := dstTypes[options.DstTypeStr]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownType, options.DstTypeStr)
	}
	options.DstType = dstType

	addrTypes := map[string]ec2.AddrType{
		"":        ec2.AddrTypeAuto,
		"private": ec2.AddrTypePrivate,
		"public":  ec2.AddrTypePublic,
		"ipv6":    ec2.AddrTypeIPv6,
	}

	addrType, ok := addrTypes[options.AddrTypeStr]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownType, options.AddrTypeStr)
	}
	options.AddrType = addrType

	return nil
}

// RunSFTP executes the SFTP intent with the given arguments.
func RunSFTP(args []string) error {
	options, err := NewSFTPOptions(args)
	if err != nil {
		return err
	}

	if options.Destination == "" {
		return ErrMissingDestination
	}

	logger := log.New(io.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	if options.Debug {
		logger.SetOutput(os.Stderr)
	}

	client, err := ec2.NewClient(options.Region, options.Profile, logger)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "ec2ssh")
	if err != nil {
		return err
	}

	defer func() { _ = os.RemoveAll(tmpDir) }()

	session, err := ssh.NewSFTPSession(client, options.DstType, options.AddrType, options.Destination,
		options.Login, options.Port, options.IdentityFile, options.UseEICE, options.EICEID,
		options.NoSendKeys, options.SFTPArgs, options.RemotePath, tmpDir, logger)
	if err != nil {
		return err
	}

	return session.Run()
}
