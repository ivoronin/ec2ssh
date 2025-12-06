package app

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/ivoronin/ec2ssh/internal/awsclient"
	"github.com/ivoronin/ec2ssh/internal/ec2client"
	"github.com/ivoronin/ec2ssh/internal/ssh"
)

// Package-level factory functions for dependency injection in tests.
// These default to the real implementations but can be overridden in tests.
var (
	loadAWSConfig   = awsclient.LoadConfig
	newEC2Client    = ec2client.NewClient
	generateKeypair = ssh.GenerateKeypair
	getPublicKey    = ssh.GetPublicKey
	executeCommand  = defaultExecuteCommand
)

// CommandRunner is a function type for executing commands.
type CommandRunner func(command string, args []string, logger *log.Logger) error

// defaultExecuteCommand is the production command executor.
func defaultExecuteCommand(command string, args []string, logger *log.Logger) error {
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logger.Printf("running %s with args: %v", command, args)

	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			logger.Printf("%s exited with code %d", command, exitError.ExitCode())
		}
		return err
	}

	logger.Printf("%s exited with code 0", command)
	return nil
}

// baseSSHSession contains common fields for all session types (SSH, SCP, SFTP).
// Fields are organized by lifecycle stage: CLI flags → parsed values → runtime state.
type baseSSHSession struct {
	// --- CLI Configuration (populated by argsieve from command-line flags) ---
	Region       string `long:"region"`
	Profile      string `long:"profile"`
	EICEID       string `long:"eice-id"`
	DstTypeStr   string `long:"destination-type"`
	AddrTypeStr  string `long:"address-type"`
	IdentityFile string `short:"i"`
	UseEICE      bool   `long:"use-eice"`
	UseSSM       bool   `long:"use-ssm"`
	NoSendKeys   bool   `long:"no-send-keys"`
	Debug        bool   `long:"debug"`

	// --- Parsed Session Parameters (set after argument parsing) ---
	DstType     ec2client.DstType  // Parsed from DstTypeStr
	AddrType    ec2client.AddrType // Parsed from AddrTypeStr
	Login       string       // SSH login user
	Destination string       // EC2 instance identifier (ID, IP, DNS, or name tag)
	Port        string       // SSH/SFTP/SCP port
	PassArgs    []string     // Passthrough args for the underlying command

	// --- Runtime State (set during run()) ---
	client          *ec2client.Client // EC2 API client
	instance        types.Instance    // Resolved EC2 instance
	destinationAddr string            // Resolved connection address
	privateKeyPath  string            // Path to SSH private key
	publicKey       string            // SSH public key content
	proxyCommand    string            // ProxyCommand for EICE/SSM tunneling
	logger          *log.Logger       // Debug logger
}

// appendOptArg appends a formatted option to args if value is non-empty.
// The format string should contain exactly one %s placeholder.
func appendOptArg(args []string, format, value string) []string {
	if value != "" {
		return append(args, fmt.Sprintf(format, value))
	}
	return args
}

// baseArgs returns common SSH options: ProxyCommand, identity file, HostKeyAlias, and passthrough args.
func (s *baseSSHSession) baseArgs() []string {
	var args []string
	args = appendOptArg(args, "-oProxyCommand=%s", s.proxyCommand)
	args = appendOptArg(args, "-i%s", s.privateKeyPath)
	args = append(args, fmt.Sprintf("-oHostKeyAlias=%s", *s.instance.InstanceId))
	args = append(args, s.PassArgs...)
	return args
}

// ParseTypes converts destination and address type strings to their enum values.
func (s *baseSSHSession) ParseTypes() error {
	var err error
	s.DstType, err = ParseDstType(s.DstTypeStr)
	if err != nil {
		return err
	}
	s.AddrType, err = ParseAddrType(s.AddrTypeStr)
	return err
}

// ApplyDefaults applies common default values.
// Sets UseEICE if EICEID is provided and defaults Login to current user.
func (s *baseSSHSession) ApplyDefaults() error {
	if s.EICEID != "" {
		s.UseEICE = true
	}

	// Validate mutually exclusive options
	if s.UseEICE && s.UseSSM {
		return fmt.Errorf("%w: --use-eice and --use-ssm", ErrExclusiveOptions)
	}

	if s.Login == "" {
		u, err := user.Current()
		if err != nil {
			return fmt.Errorf("unable to determine current user: %w", err)
		}
		s.Login = u.Username
	}

	return nil
}

// initLogger initializes the debug logger based on the Debug flag.
func (s *baseSSHSession) initLogger() {
	s.logger = log.New(io.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	if s.Debug {
		s.logger.SetOutput(os.Stderr)
	}
}

func (s *baseSSHSession) setupSSHKeys(tmpDir string) error {
	var err error

	if s.IdentityFile == "" {
		s.privateKeyPath, s.publicKey, err = generateKeypair(tmpDir)
		if err != nil {
			return fmt.Errorf("unable to generate ephemeral SSH keypair: %w", err)
		}
	} else {
		s.privateKeyPath = s.IdentityFile
		s.publicKey, err = getPublicKey(s.IdentityFile)
		if err != nil {
			return fmt.Errorf("unable to read public key from %s: %w", s.IdentityFile, err)
		}
	}

	return nil
}

// sendSSHPublicKey sends the public key to the instance via EC2 Instance Connect.
func (s *baseSSHSession) sendSSHPublicKey() error {
	if err := s.client.SendSSHPublicKey(s.instance, s.Login, s.publicKey); err != nil {
		return fmt.Errorf("unable to send SSH public key: %w", err)
	}
	return nil
}

// setupProxyCommand configures the SSH ProxyCommand for EICE or SSM tunneling.
// Uses %p for port substitution by SSH.
func (s *baseSSHSession) setupProxyCommand() error {
	args := []string{os.Args[0]}

	if s.UseSSM {
		args = append(args, "--ssm-tunnel")
		args = append(args, "--instance-id", *s.instance.InstanceId)
		args = append(args, "--port", "%p")
	} else if s.UseEICE {
		// Resolve EICE ID if not explicitly provided
		eiceID := s.EICEID
		if eiceID == "" {
			eice, err := s.client.GuessEICEByVPCAndSubnet(*s.instance.VpcId, *s.instance.SubnetId)
			if err != nil {
				return fmt.Errorf("unable to find EICE endpoint: %w", err)
			}
			eiceID = *eice.InstanceConnectEndpointId
		}

		args = append(args, "--eice-tunnel")
		args = append(args, "--host", *s.instance.PrivateIpAddress)
		args = append(args, "--port", "%p")
		args = append(args, "--eice-id", eiceID)
	}

	if s.Region != "" {
		args = append(args, "--region", s.Region)
	}
	if s.Profile != "" {
		args = append(args, "--profile", s.Profile)
	}
	if s.Debug {
		args = append(args, "--debug")
	}

	s.proxyCommand = strings.Join(args, " ")
	return nil
}


// run executes the session command. Called by embedded types.
// buildArgs is called after setup completes, ensuring runtime fields are populated.
func (s *baseSSHSession) run(command string, buildArgs func() []string) error {
	// Initialize logger
	s.initLogger()

	// Load AWS config
	cfg, err := loadAWSConfig(s.Region, s.Profile, s.logger)
	if err != nil {
		return err
	}

	// Create EC2 client
	s.client, err = newEC2Client(cfg, s.logger)
	if err != nil {
		return err
	}

	// Get instance
	s.instance, err = s.client.GetInstance(s.DstType, s.Destination)
	if err != nil {
		return fmt.Errorf("unable to get instance: %w", err)
	}

	// Sanity check: AWS API should always return InstanceId, but panic with
	// a helpful message rather than a cryptic nil pointer dereference if it doesn't
	if s.instance.InstanceId == nil {
		panic("ec2ssh: AWS returned instance without InstanceId - this should never happen")
	}

	// Create temp dir for ephemeral keys
	tmpDir, err := os.MkdirTemp("", "ec2ssh")
	if err != nil {
		return fmt.Errorf("unable to create temp directory for SSH keys: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Setup SSH keys
	if err := s.setupSSHKeys(tmpDir); err != nil {
		return err
	}

	// Send SSH public key
	if !s.NoSendKeys {
		if err := s.sendSSHPublicKey(); err != nil {
			return err
		}
	}

	// Setup destination address and proxy command (EICE or SSM)
	if s.UseEICE || s.UseSSM {
		s.destinationAddr = *s.instance.InstanceId
		if err := s.setupProxyCommand(); err != nil {
			return err
		}
	} else {
		s.destinationAddr, err = ec2client.GetInstanceAddr(s.instance, s.AddrType)
		if err != nil {
			return err
		}
	}

	return executeCommand(command, buildArgs(), s.logger)
}
