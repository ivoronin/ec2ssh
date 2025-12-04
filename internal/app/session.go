package app

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/ivoronin/ec2ssh/internal/ec2"
	"github.com/ivoronin/ec2ssh/internal/ssh"
)

// Package-level factory functions for dependency injection in tests.
// These default to the real implementations but can be overridden in tests.
var (
	newEC2Client    = ec2.NewClient
	generateKeypar  = ssh.GenerateKeypair
	getPublicKey    = ssh.GetPublicKey
	executeCommand  = defaultExecuteCommand
)

// CommandRunner is a function type for executing commands.
type CommandRunner func(command string, args []string, tunnelURI string, logger *log.Logger) error

// defaultExecuteCommand is the production command executor.
func defaultExecuteCommand(command string, args []string, tunnelURI string, logger *log.Logger) error {
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if tunnelURI != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("EC2SSH_TUNNEL_URI=%s", tunnelURI))
	}

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

// baseSession contains common fields for all session types (SSH, SCP, SFTP).
// Fields are organized by lifecycle stage: CLI flags → parsed values → runtime state.
type baseSession struct {
	// --- CLI Configuration (populated by argsieve from command-line flags) ---
	Region       string `long:"region"`
	Profile      string `long:"profile"`
	EICEID       string `long:"eice-id"`
	DstTypeStr   string `long:"destination-type"`
	AddrTypeStr  string `long:"address-type"`
	IdentityFile string `short:"i"`
	UseEICE      bool   `long:"use-eice"`
	NoSendKeys   bool   `long:"no-send-keys"`
	Debug        bool   `long:"debug"`

	// --- Parsed Session Parameters (set after argument parsing) ---
	DstType     ec2.DstType  // Parsed from DstTypeStr
	AddrType    ec2.AddrType // Parsed from AddrTypeStr
	Login       string       // SSH login user
	Destination string       // EC2 instance identifier (ID, IP, DNS, or name tag)
	Port        string       // SSH/SFTP/SCP port
	PassArgs    []string     // Passthrough args for the underlying command

	// --- Runtime State (set during run()) ---
	client          *ec2.Client    // EC2 API client
	instance        types.Instance // Resolved EC2 instance
	destinationAddr string         // Resolved connection address
	privateKeyPath  string         // Path to SSH private key
	publicKey       string         // SSH public key content
	proxyCommand    string         // ProxyCommand for EICE tunneling
	tunnelURI       string         // EICE tunnel URI
	logger          *log.Logger    // Debug logger
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
func (s *baseSession) baseArgs() []string {
	var args []string
	args = appendOptArg(args, "-oProxyCommand=%s", s.proxyCommand)
	args = appendOptArg(args, "-i%s", s.privateKeyPath)
	args = append(args, fmt.Sprintf("-oHostKeyAlias=%s", *s.instance.InstanceId))
	args = append(args, s.PassArgs...)
	return args
}

// ParseTypes converts destination and address type strings to their enum values.
func (s *baseSession) ParseTypes() error {
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
func (s *baseSession) ApplyDefaults() error {
	if s.EICEID != "" {
		s.UseEICE = true
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
func (s *baseSession) initLogger() {
	s.logger = log.New(io.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	if s.Debug {
		s.logger.SetOutput(os.Stderr)
	}
}

func (s *baseSession) setupSSHKeys(tmpDir string) error {
	var err error

	if s.IdentityFile == "" {
		s.privateKeyPath, s.publicKey, err = generateKeypar(tmpDir)
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
func (s *baseSession) sendSSHPublicKey() error {
	if err := s.client.SendSSHPublicKey(s.instance, s.Login, s.publicKey); err != nil {
		return fmt.Errorf("unable to send SSH public key: %w", err)
	}
	return nil
}

// setupTunnel configures EICE: overrides destination address, sets proxy command, and creates tunnel URI.
func (s *baseSession) setupTunnel() error {
	s.proxyCommand = fmt.Sprintf("%s --wscat", os.Args[0])

	tunnelURI, err := s.client.CreateEICETunnelURI(s.instance, s.Port, s.EICEID)
	if err != nil {
		return fmt.Errorf("unable to setup EICE tunnel: %w", err)
	}
	s.tunnelURI = tunnelURI
	return nil
}


// run executes the session command. Called by embedded types.
// buildArgs is called after setup completes, ensuring runtime fields are populated.
func (s *baseSession) run(command string, buildArgs func() []string) error {
	// Initialize logger
	s.initLogger()

	// Create EC2 client
	var err error
	s.client, err = newEC2Client(s.Region, s.Profile, s.logger)
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

	// Setup destination address and EICE tunnel
	if s.UseEICE {
		s.destinationAddr = *s.instance.InstanceId
		if err := s.setupTunnel(); err != nil {
			return err
		}
	} else {
		s.destinationAddr, err = ec2.GetInstanceAddr(s.instance, s.AddrType)
		if err != nil {
			return err
		}
	}

	return executeCommand(command, buildArgs(), s.tunnelURI, s.logger)
}
