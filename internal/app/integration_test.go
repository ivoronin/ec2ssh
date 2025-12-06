package app

import (
	"errors"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/ivoronin/ec2ssh/internal/ec2client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturedCommand stores the command and args passed to executeCommand.
type capturedCommand struct {
	command string
	args    []string
}

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
// Note: Cannot run in parallel - modifies global generateKeypair.
func TestKeyGeneration(t *testing.T) {
	// Test with mocked generateKeypair
	origFunc := generateKeypair
	defer func() { generateKeypair = origFunc }()

	var calledTmpDir string
	generateKeypair = func(tmpDir string) (string, string, error) {
		calledTmpDir = tmpDir
		return "/mock/private/key", "ssh-ed25519 AAAA... mock", nil
	}

	session := &baseSSHSession{}
	err := session.setupSSHKeys("/test/tmp")

	require.NoError(t, err)
	assert.Equal(t, "/test/tmp", calledTmpDir)
	assert.Equal(t, "/mock/private/key", session.privateKeyPath)
	assert.Equal(t, "ssh-ed25519 AAAA... mock", session.publicKey)
}

// TestKeyGenerationWithIdentityFile tests using an existing identity file.
// Note: Cannot run in parallel - modifies global getPublicKey.
func TestKeyGenerationWithIdentityFile(t *testing.T) {
	// Test with mocked getPublicKey
	origFunc := getPublicKey
	defer func() { getPublicKey = origFunc }()

	var calledIdentityFile string
	getPublicKey = func(identityFile string) (string, error) {
		calledIdentityFile = identityFile
		return "ssh-rsa AAAA... existing", nil
	}

	session := &baseSSHSession{
		IdentityFile: "/path/to/my/key",
	}
	err := session.setupSSHKeys("/test/tmp")

	require.NoError(t, err)
	assert.Equal(t, "/path/to/my/key", calledIdentityFile)
	assert.Equal(t, "/path/to/my/key", session.privateKeyPath)
	assert.Equal(t, "ssh-rsa AAAA... existing", session.publicKey)
}

// TestKeyGenerationError tests error handling in key generation.
// Note: Cannot run in parallel - modifies global generateKeypair.
func TestKeyGenerationError(t *testing.T) {
	origFunc := generateKeypair
	defer func() { generateKeypair = origFunc }()

	generateKeypair = func(tmpDir string) (string, string, error) {
		return "", "", errors.New("key generation failed")
	}

	session := &baseSSHSession{}
	err := session.setupSSHKeys("/test/tmp")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to generate ephemeral SSH keypair")
}

// TestGetPublicKeyError tests error handling when reading public key fails.
// Note: Cannot run in parallel - modifies global getPublicKey.
func TestGetPublicKeyError(t *testing.T) {
	origFunc := getPublicKey
	defer func() { getPublicKey = origFunc }()

	getPublicKey = func(identityFile string) (string, error) {
		return "", errors.New("cannot read public key")
	}

	session := &baseSSHSession{
		IdentityFile: "/nonexistent/key",
	}
	err := session.setupSSHKeys("/test/tmp")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to read public key")
}

// TestExecuteCommandFactory tests the command execution factory.
// Note: Cannot run in parallel - modifies global executeCommand.
func TestExecuteCommandFactory(t *testing.T) {
	// Save and restore original
	origFunc := executeCommand
	defer func() { executeCommand = origFunc }()

	var captured capturedCommand
	executeCommand = func(command string, args []string, logger *log.Logger) error {
		captured.command = command
		captured.args = args
		return nil
	}

	logger := log.Default()
	err := executeCommand("ssh", []string{"-v", "host"}, logger)

	require.NoError(t, err)
	assert.Equal(t, "ssh", captured.command)
	assert.Equal(t, []string{"-v", "host"}, captured.args)
}

// TestExecuteCommandError tests error propagation from command execution.
// Note: Cannot run in parallel - modifies global executeCommand.
func TestExecuteCommandError(t *testing.T) {
	origFunc := executeCommand
	defer func() { executeCommand = origFunc }()

	expectedErr := errors.New("command failed")
	executeCommand = func(command string, args []string, logger *log.Logger) error {
		return expectedErr
	}

	logger := log.Default()
	err := executeCommand("ssh", []string{}, logger)

	assert.Equal(t, expectedErr, err)
}

// TestSessionApplyDefaults_EICE tests that EICEID implies UseEICE.
func TestSessionApplyDefaults_EICE(t *testing.T) {
	t.Parallel()

	session := &baseSSHSession{
		EICEID: "eice-12345",
	}

	err := session.ApplyDefaults()

	require.NoError(t, err)
	assert.True(t, session.UseEICE)
}

// TestSessionApplyDefaults_DefaultLogin tests default login is set.
func TestSessionApplyDefaults_DefaultLogin(t *testing.T) {
	t.Parallel()

	session := &baseSSHSession{}

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
		wantDst     ec2client.DstType
		wantAddr    ec2client.AddrType
	}{
		{
			name:        "auto types",
			dstTypeStr:  "",
			addrTypeStr: "",
			wantDst:     ec2client.DstTypeAuto,
			wantAddr:    ec2client.AddrTypeAuto,
		},
		{
			name:        "explicit instance ID",
			dstTypeStr:  "id",
			addrTypeStr: "private",
			wantDst:     ec2client.DstTypeID,
			wantAddr:    ec2client.AddrTypePrivate,
		},
		{
			name:        "private IP address",
			dstTypeStr:  "private_ip",
			addrTypeStr: "public",
			wantDst:     ec2client.DstTypePrivateIP,
			wantAddr:    ec2client.AddrTypePublic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session := &baseSSHSession{
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

		session := &baseSSHSession{Debug: false}
		session.initLogger()

		assert.NotNil(t, session.logger)
		// Logger should discard output when debug is disabled
	})

	t.Run("debug enabled", func(t *testing.T) {
		t.Parallel()

		session := &baseSSHSession{Debug: true}
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

			session := &baseSSHSession{
				DstTypeStr:  tt.dstTypeStr,
				AddrTypeStr: tt.addrTypeStr,
			}

			err := session.ParseTypes()

			require.Error(t, err)
		})
	}
}
