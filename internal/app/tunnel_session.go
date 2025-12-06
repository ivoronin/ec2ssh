package app

import (
	"errors"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/ivoronin/ec2ssh/internal/cli/argsieve"
	"github.com/ivoronin/ec2ssh/internal/tunnel"
	"github.com/mmmorris1975/ssm-session-client/ssmclient"
)

// Errors for tunnel session validation.
var (
	ErrMissingHost       = errors.New("missing required --host")
	ErrMissingEICEID     = errors.New("missing required --eice-id")
	ErrMissingInstanceID = errors.New("missing required --instance-id")
	ErrMissingPort       = errors.New("missing required --port")
)

// baseTunnelSession contains common fields for tunnel sessions.
type baseTunnelSession struct {
	// CLI flags (parsed by argsieve)
	Region  string `long:"region"`
	Profile string `long:"profile"`
	Debug   bool   `long:"debug"`
	Port    string `long:"port"`

	// Runtime state
	logger *log.Logger
}

// initLogger initializes the debug logger based on the Debug flag.
func (s *baseTunnelSession) initLogger() {
	s.logger = log.New(io.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	if s.Debug {
		s.logger.SetOutput(os.Stderr)
	}
}

// EICETunnelSession handles EICE WebSocket tunnel connections.
type EICETunnelSession struct {
	baseTunnelSession
	Host   string `long:"host"`
	EICEID string `long:"eice-id"`
}

// NewEICETunnelSession creates an EICETunnelSession from command-line arguments.
func NewEICETunnelSession(args []string) (*EICETunnelSession, error) {
	var session EICETunnelSession

	sieve := argsieve.New(&session, nil)
	_, _, err := sieve.Sift(args)
	if err != nil {
		return nil, err
	}

	// Validate required fields
	if session.Host == "" {
		return nil, ErrMissingHost
	}
	if session.EICEID == "" {
		return nil, ErrMissingEICEID
	}
	if session.Port == "" {
		return nil, ErrMissingPort
	}

	return &session, nil
}

// Run executes the EICE tunnel session.
func (s *EICETunnelSession) Run() error {
	s.initLogger()

	// Load AWS config
	cfg, err := loadAWSConfig(s.Region, s.Profile, s.logger)
	if err != nil {
		return err
	}

	// Create EC2 client for EICE lookup and signing
	client, err := newEC2Client(cfg, s.logger)
	if err != nil {
		return err
	}

	// Create signed tunnel URI
	uri, err := client.CreateEICETunnelURI(s.Host, s.Port, s.EICEID)
	if err != nil {
		return err
	}

	s.logger.Printf("connecting to EICE tunnel: %s", s.Host)

	// Open WebSocket and pipe I/O
	return tunnel.Run(uri)
}

// SSMTunnelSession handles SSM tunnel connections.
type SSMTunnelSession struct {
	baseTunnelSession
	InstanceID string `long:"instance-id"`
}

// NewSSMTunnelSession creates an SSMTunnelSession from command-line arguments.
func NewSSMTunnelSession(args []string) (*SSMTunnelSession, error) {
	var session SSMTunnelSession

	sieve := argsieve.New(&session, nil)
	_, _, err := sieve.Sift(args)
	if err != nil {
		return nil, err
	}

	// Validate required fields
	if session.InstanceID == "" {
		return nil, ErrMissingInstanceID
	}
	if session.Port == "" {
		return nil, ErrMissingPort
	}

	return &session, nil
}

// Run executes the SSM tunnel session.
func (s *SSMTunnelSession) Run() error {
	s.initLogger()

	// Parse port
	port, err := strconv.Atoi(s.Port)
	if err != nil {
		return err
	}

	// Load AWS config
	cfg, err := loadAWSConfig(s.Region, s.Profile, s.logger)
	if err != nil {
		return err
	}

	s.logger.Printf("connecting to SSM tunnel: %s", s.InstanceID)

	return ssmclient.SSHSession(cfg, &ssmclient.PortForwardingInput{
		Target:     s.InstanceID,
		RemotePort: port,
	})
}
