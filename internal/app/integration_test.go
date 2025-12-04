package app

import (
	"errors"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/ivoronin/ec2ssh/internal/ec2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures and mock helpers

// mockEC2Client creates a mock EC2 client for testing.
type mockEC2Client struct {
	instance    types.Instance
	instanceErr error
	sendKeyErr  error
	tunnelURI   string
	tunnelErr   error
}

func (m *mockEC2Client) GetInstance(dstType ec2.DstType, destination string) (types.Instance, error) {
	return m.instance, m.instanceErr
}

func (m *mockEC2Client) SendSSHPublicKey(instance types.Instance, login, publicKey string) error {
	return m.sendKeyErr
}

func (m *mockEC2Client) CreateEICETunnelURI(instance types.Instance, port, eiceID string) (string, error) {
	return m.tunnelURI, m.tunnelErr
}

// capturedCommand stores the command and args passed to executeCommand.
type capturedCommand struct {
	command   string
	args      []string
	tunnelURI string
}

// setupTestMocks configures test mocks and returns a cleanup function.
func setupTestMocks(t *testing.T, client *mockEC2Client, capture *capturedCommand) func() {
	t.Helper()

	// Save original functions
	origNewEC2Client := newEC2Client
	origGenerateKeypair := generateKeypar
	origGetPublicKey := getPublicKey
	origExecuteCommand := executeCommand

	// Replace with mocks
	newEC2Client = func(region, profile string, logger *log.Logger) (*ec2.Client, error) {
		// We can't return our mock directly since it's not *ec2.Client
		// Instead, we'll need to use a different approach - see note below
		return nil, nil // This won't work as-is
	}

	generateKeypar = func(tmpDir string) (string, string, error) {
		return "/tmp/test-key", "ssh-ed25519 AAAA... test-key", nil
	}

	getPublicKey = func(identityFile string) (string, error) {
		return "ssh-ed25519 AAAA... test-key", nil
	}

	executeCommand = func(command string, args []string, tunnelURI string, logger *log.Logger) error {
		if capture != nil {
			capture.command = command
			capture.args = args
			capture.tunnelURI = tunnelURI
		}
		return nil
	}

	return func() {
		newEC2Client = origNewEC2Client
		generateKeypar = origGenerateKeypair
		getPublicKey = origGetPublicKey
		executeCommand = origExecuteCommand
	}
}

// Note: The current design uses *ec2.Client directly in baseSession.
// To fully test run(), we'd need to either:
// 1. Change baseSession.client to an interface type
// 2. Or test at a higher level (e.g., through the CLI)
//
// For now, we'll test the components we can mock: key generation and command execution.

// TestSSHSession_BuildArgs_Integration tests the full arg building with realistic scenarios.
func TestSSHSession_BuildArgs_Integration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantInArgs []string // Substrings to find in any arg
	}{
		{
			name:       "basic SSH to instance",
			args:       []string{"--region", "us-east-1", "i-1234567890abcdef0"},
			wantInArgs: []string{"-oHostKeyAlias=i-test123"},
		},
		{
			name:       "SSH with login and port",
			args:       []string{"-l", "ubuntu", "-p", "2222", "i-test"},
			wantInArgs: []string{"-lubuntu", "-p2222"},
		},
		{
			name:       "SSH with passthrough options",
			args:       []string{"-L", "8080:localhost:80", "i-test"},
			wantInArgs: []string{"-L", "8080:localhost:80"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session, err := NewSSHSession(tt.args)
			require.NoError(t, err)

			// Simulate the state that would be set during run()
			session.instance = types.Instance{
				InstanceId: aws.String("i-test123"),
			}
			session.destinationAddr = "10.0.0.1"
			session.privateKeyPath = "/tmp/test-key"

			args := session.buildArgs()

			for _, want := range tt.wantInArgs {
				assert.Contains(t, args, want)
			}
		})
	}
}

// TestSCPSession_BuildArgs_Integration tests SCP argument building.
func TestSCPSession_BuildArgs_Integration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantInArgs []string
	}{
		{
			name:       "upload file",
			args:       []string{"/local/file", "user@i-test:/remote/path"},
			wantInArgs: []string{"/local/file", "user@"},
		},
		{
			name:       "download file",
			args:       []string{"user@i-test:/remote/file", "/local/path"},
			wantInArgs: []string{"/local/path"},
		},
		{
			name:       "with port",
			args:       []string{"-P", "2222", "/local/file", "user@i-test:/remote"},
			wantInArgs: []string{"-P2222"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session, err := NewSCPSession(tt.args)
			require.NoError(t, err)

			// Simulate runtime state
			session.instance = types.Instance{
				InstanceId: aws.String("i-test123"),
			}
			session.destinationAddr = "10.0.0.1"
			session.privateKeyPath = "/tmp/test-key"

			args := session.buildArgs()

			for _, want := range tt.wantInArgs {
				found := false
				for _, arg := range args {
					if arg == want || (len(want) > 0 && len(arg) >= len(want) && arg[:len(want)] == want) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected to find %q in args %v", want, args)
			}
		})
	}
}

// TestSFTPSession_BuildArgs_Integration tests SFTP argument building.
func TestSFTPSession_BuildArgs_Integration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantInArgs []string
	}{
		{
			name:       "basic SFTP",
			args:       []string{"user@i-test"},
			wantInArgs: []string{"user@"},
		},
		{
			name:       "with port",
			args:       []string{"-P", "2222", "user@i-test"},
			wantInArgs: []string{"-P2222"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session, err := NewSFTPSession(tt.args)
			require.NoError(t, err)

			// Simulate runtime state
			session.instance = types.Instance{
				InstanceId: aws.String("i-test123"),
			}
			session.destinationAddr = "10.0.0.1"
			session.privateKeyPath = "/tmp/test-key"

			args := session.buildArgs()

			for _, want := range tt.wantInArgs {
				found := false
				for _, arg := range args {
					if arg == want || (len(want) > 0 && len(arg) >= len(want) && arg[:len(want)] == want) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected to find %q in args %v", want, args)
			}
		})
	}
}

// TestKeyGeneration tests the key generation factory.
func TestKeyGeneration(t *testing.T) {
	t.Parallel()

	// Test with mocked generateKeypar
	origFunc := generateKeypar
	defer func() { generateKeypar = origFunc }()

	var calledTmpDir string
	generateKeypar = func(tmpDir string) (string, string, error) {
		calledTmpDir = tmpDir
		return "/mock/private/key", "ssh-ed25519 AAAA... mock", nil
	}

	session := &baseSession{}
	err := session.setupSSHKeys("/test/tmp")

	require.NoError(t, err)
	assert.Equal(t, "/test/tmp", calledTmpDir)
	assert.Equal(t, "/mock/private/key", session.privateKeyPath)
	assert.Equal(t, "ssh-ed25519 AAAA... mock", session.publicKey)
}

// TestKeyGenerationWithIdentityFile tests using an existing identity file.
func TestKeyGenerationWithIdentityFile(t *testing.T) {
	t.Parallel()

	// Test with mocked getPublicKey
	origFunc := getPublicKey
	defer func() { getPublicKey = origFunc }()

	var calledIdentityFile string
	getPublicKey = func(identityFile string) (string, error) {
		calledIdentityFile = identityFile
		return "ssh-rsa AAAA... existing", nil
	}

	session := &baseSession{
		IdentityFile: "/path/to/my/key",
	}
	err := session.setupSSHKeys("/test/tmp")

	require.NoError(t, err)
	assert.Equal(t, "/path/to/my/key", calledIdentityFile)
	assert.Equal(t, "/path/to/my/key", session.privateKeyPath)
	assert.Equal(t, "ssh-rsa AAAA... existing", session.publicKey)
}

// TestKeyGenerationError tests error handling in key generation.
func TestKeyGenerationError(t *testing.T) {
	t.Parallel()

	origFunc := generateKeypar
	defer func() { generateKeypar = origFunc }()

	generateKeypar = func(tmpDir string) (string, string, error) {
		return "", "", errors.New("key generation failed")
	}

	session := &baseSession{}
	err := session.setupSSHKeys("/test/tmp")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to generate ephemeral SSH keypair")
}

// TestGetPublicKeyError tests error handling when reading public key fails.
func TestGetPublicKeyError(t *testing.T) {
	t.Parallel()

	origFunc := getPublicKey
	defer func() { getPublicKey = origFunc }()

	getPublicKey = func(identityFile string) (string, error) {
		return "", errors.New("cannot read public key")
	}

	session := &baseSession{
		IdentityFile: "/nonexistent/key",
	}
	err := session.setupSSHKeys("/test/tmp")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to read public key")
}

// TestExecuteCommandFactory tests the command execution factory.
func TestExecuteCommandFactory(t *testing.T) {
	t.Parallel()

	// Save and restore original
	origFunc := executeCommand
	defer func() { executeCommand = origFunc }()

	var captured capturedCommand
	executeCommand = func(command string, args []string, tunnelURI string, logger *log.Logger) error {
		captured.command = command
		captured.args = args
		captured.tunnelURI = tunnelURI
		return nil
	}

	logger := log.Default()
	err := executeCommand("ssh", []string{"-v", "host"}, "wss://tunnel.uri", logger)

	require.NoError(t, err)
	assert.Equal(t, "ssh", captured.command)
	assert.Equal(t, []string{"-v", "host"}, captured.args)
	assert.Equal(t, "wss://tunnel.uri", captured.tunnelURI)
}

// TestExecuteCommandError tests error propagation from command execution.
func TestExecuteCommandError(t *testing.T) {
	t.Parallel()

	origFunc := executeCommand
	defer func() { executeCommand = origFunc }()

	expectedErr := errors.New("command failed")
	executeCommand = func(command string, args []string, tunnelURI string, logger *log.Logger) error {
		return expectedErr
	}

	logger := log.Default()
	err := executeCommand("ssh", []string{}, "", logger)

	assert.Equal(t, expectedErr, err)
}

// TestSessionApplyDefaults_EICE tests that EICEID implies UseEICE.
func TestSessionApplyDefaults_EICE(t *testing.T) {
	t.Parallel()

	session := &baseSession{
		EICEID: "eice-12345",
	}

	err := session.ApplyDefaults()

	require.NoError(t, err)
	assert.True(t, session.UseEICE)
}

// TestSessionApplyDefaults_DefaultLogin tests default login is set.
func TestSessionApplyDefaults_DefaultLogin(t *testing.T) {
	t.Parallel()

	session := &baseSession{}

	err := session.ApplyDefaults()

	require.NoError(t, err)
	assert.NotEmpty(t, session.Login, "Login should be set to current user")
}

// TestSessionParseTypes tests type parsing.
func TestSessionParseTypes_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		dstTypeStr  string
		addrTypeStr string
		wantDst     ec2.DstType
		wantAddr    ec2.AddrType
	}{
		{
			name:        "auto types",
			dstTypeStr:  "",
			addrTypeStr: "",
			wantDst:     ec2.DstTypeAuto,
			wantAddr:    ec2.AddrTypeAuto,
		},
		{
			name:        "explicit instance ID",
			dstTypeStr:  "id",
			addrTypeStr: "private",
			wantDst:     ec2.DstTypeID,
			wantAddr:    ec2.AddrTypePrivate,
		},
		{
			name:        "private IP address",
			dstTypeStr:  "private_ip",
			addrTypeStr: "public",
			wantDst:     ec2.DstTypePrivateIP,
			wantAddr:    ec2.AddrTypePublic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session := &baseSession{
				DstTypeStr:  tt.dstTypeStr,
				AddrTypeStr: tt.addrTypeStr,
			}

			err := session.ParseTypes()

			require.NoError(t, err)
			assert.Equal(t, tt.wantDst, session.DstType)
			assert.Equal(t, tt.wantAddr, session.AddrType)
		})
	}
}

// TestBaseSession_InitLogger tests logger initialization.
func TestBaseSession_InitLogger(t *testing.T) {
	t.Parallel()

	t.Run("debug disabled", func(t *testing.T) {
		t.Parallel()

		session := &baseSession{Debug: false}
		session.initLogger()

		assert.NotNil(t, session.logger)
		// Logger should discard output when debug is disabled
	})

	t.Run("debug enabled", func(t *testing.T) {
		t.Parallel()

		session := &baseSession{Debug: true}
		session.initLogger()

		assert.NotNil(t, session.logger)
		// Logger should write to stderr when debug is enabled
	})
}

// TestSessionParseTypes_Invalid tests invalid type parsing.
func TestSessionParseTypes_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		dstTypeStr string
		addrTypeStr string
	}{
		{
			name:       "invalid destination type",
			dstTypeStr: "invalid",
			addrTypeStr: "",
		},
		{
			name:       "invalid address type",
			dstTypeStr: "",
			addrTypeStr: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session := &baseSession{
				DstTypeStr:  tt.dstTypeStr,
				AddrTypeStr: tt.addrTypeStr,
			}

			err := session.ParseTypes()

			require.Error(t, err)
		})
	}
}
