package app

import (
	"errors"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
	"github.com/ivoronin/ec2ssh/internal/ec2client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Mock Type Aliases - Using shared mocks from ec2client.testing.go
// =============================================================================

// Use the exported mock types from ec2client package to avoid duplication
type mockEC2API = ec2client.MockEC2API
type mockEC2InstanceConnectAPI = ec2client.MockEC2InstanceConnectAPI
type mockHTTPRequestSigner = ec2client.MockHTTPRequestSigner

// =============================================================================
// Test Fixtures and Helpers
// =============================================================================

// testInstance is a standard test instance fixture
var testInstance = types.Instance{
	InstanceId:       aws.String("i-1234567890abcdef0"),
	PrivateIpAddress: aws.String("10.0.0.1"),
	PublicIpAddress:  aws.String("52.1.2.3"),
	VpcId:            aws.String("vpc-123"),
	SubnetId:         aws.String("subnet-456"),
	State:            &types.InstanceState{Name: types.InstanceStateNameRunning},
}

// commandCapture holds captured command execution details
type commandCapture struct {
	command string
	args    []string
}

// setupMocksForRun sets up all DI mocks for a Run() test and returns cleanup function
func setupMocksForRun(t *testing.T, instance types.Instance, captureCmd *commandCapture) (*mockEC2API, *mockEC2InstanceConnectAPI) {
	t.Helper()

	// Save originals
	origLoadAWSConfig := loadAWSConfig
	origNewEC2Client := newEC2Client
	origGenerateKeypair := generateKeypair
	origExecuteCommand := executeCommand

	// Cleanup
	t.Cleanup(func() {
		loadAWSConfig = origLoadAWSConfig
		newEC2Client = origNewEC2Client
		generateKeypair = origGenerateKeypair
		executeCommand = origExecuteCommand
	})

	// Mock AWS config loading
	loadAWSConfig = func(region, profile string, logger *log.Logger) (aws.Config, error) {
		return aws.Config{Region: "us-east-1"}, nil
	}

	// Create mock EC2 clients
	ec2Mock := new(mockEC2API)
	connectMock := new(mockEC2InstanceConnectAPI)

	// Setup mock to return test instance
	ec2Mock.On("DescribeInstances", mock.Anything, mock.Anything).Return(
		&ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{{Instances: []types.Instance{instance}}},
		}, nil,
	)

	// Setup mock for SendSSHPublicKey
	connectMock.On("SendSSHPublicKey", mock.Anything, mock.Anything).Return(
		&ec2instanceconnect.SendSSHPublicKeyOutput{Success: true}, nil,
	)

	// Mock EC2 client creation
	newEC2Client = func(cfg aws.Config, logger *log.Logger) (*ec2client.Client, error) {
		return ec2client.NewTestClient(ec2Mock, connectMock, new(mockHTTPRequestSigner)), nil
	}

	// Mock keypair generation
	generateKeypair = func(tmpDir string) (string, string, error) {
		return "/tmp/test_key", "ssh-ed25519 AAAAC3NzaC1... test@host", nil
	}

	// Mock command execution - capture args
	executeCommand = func(cmd string, args []string, logger *log.Logger) error {
		if captureCmd != nil {
			captureCmd.command = cmd
			captureCmd.args = args
		}
		return nil
	}

	return ec2Mock, connectMock
}

// =============================================================================
// SSHSession.Run() Integration Tests
// =============================================================================

func TestSSHSession_Run_Success(t *testing.T) {
	// No t.Parallel() - modifies global DI variables

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSSHSession([]string{"i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	// Verify ssh was called
	assert.Equal(t, "ssh", captured.command)
	// Verify instance ID is used in HostKeyAlias
	assert.Contains(t, captured.args, "-oHostKeyAlias=i-1234567890abcdef0")
	// Verify identity file is set
	assert.Contains(t, captured.args, "-i/tmp/test_key")
	// Verify destination is the private IP (default)
	assert.Equal(t, "10.0.0.1", captured.args[len(captured.args)-1])
}

func TestSSHSession_Run_WithLogin(t *testing.T) {
	// No t.Parallel() - modifies global DI vars
	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSSHSession([]string{"-l", "ec2-user", "i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	assert.Equal(t, "ssh", captured.command)
	assert.Contains(t, captured.args, "-lec2-user")
}

func TestSSHSession_Run_WithPort(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSSHSession([]string{"-p", "2222", "i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	assert.Equal(t, "ssh", captured.command)
	// Port flag is passthrough: appears as separate args "-p" and "2222"
	assert.Contains(t, captured.args, "-p")
	assert.Contains(t, captured.args, "2222")
}

func TestSSHSession_Run_WithSSMProxy(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSSHSession([]string{"--use-ssm", "i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	// Verify ProxyCommand contains --ssm-tunnel
	foundProxy := false
	for _, arg := range captured.args {
		if len(arg) > 15 && arg[:15] == "-oProxyCommand=" {
			assert.Contains(t, arg, "--ssm-tunnel")
			assert.Contains(t, arg, "--instance-id")
			assert.Contains(t, arg, "i-1234567890abcdef0")
			foundProxy = true
		}
	}
	assert.True(t, foundProxy, "ProxyCommand should be set for SSM")

	// Destination should be instance ID when using proxy
	assert.Equal(t, "i-1234567890abcdef0", captured.args[len(captured.args)-1])
}

func TestSSHSession_Run_WithEICEProxy(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	// Use --eice-id to avoid EICE lookup
	session, err := NewSSHSession([]string{"--eice-id", "eice-123", "i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	// Verify ProxyCommand contains --eice-tunnel
	foundProxy := false
	for _, arg := range captured.args {
		if len(arg) > 15 && arg[:15] == "-oProxyCommand=" {
			assert.Contains(t, arg, "--eice-tunnel")
			assert.Contains(t, arg, "--eice-id")
			assert.Contains(t, arg, "eice-123")
			foundProxy = true
		}
	}
	assert.True(t, foundProxy, "ProxyCommand should be set for EICE")
}

func TestSSHSession_Run_WithNoSendKeys(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	var captured commandCapture
	_, connectMock := setupMocksForRun(t, testInstance, &captured)

	session, err := NewSSHSession([]string{"--no-send-keys", "i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	// SendSSHPublicKey should NOT be called
	connectMock.AssertNotCalled(t, "SendSSHPublicKey", mock.Anything, mock.Anything)
}

func TestSSHSession_Run_WithIdentityFile(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	// Save originals
	origLoadAWSConfig := loadAWSConfig
	origNewEC2Client := newEC2Client
	origGetPublicKey := getPublicKey
	origExecuteCommand := executeCommand

	t.Cleanup(func() {
		loadAWSConfig = origLoadAWSConfig
		newEC2Client = origNewEC2Client
		getPublicKey = origGetPublicKey
		executeCommand = origExecuteCommand
	})

	// Mock AWS config
	loadAWSConfig = func(region, profile string, logger *log.Logger) (aws.Config, error) {
		return aws.Config{Region: "us-east-1"}, nil
	}

	// Mock EC2 client
	ec2Mock := new(mockEC2API)
	connectMock := new(mockEC2InstanceConnectAPI)

	ec2Mock.On("DescribeInstances", mock.Anything, mock.Anything).Return(
		&ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{{Instances: []types.Instance{testInstance}}},
		}, nil,
	)
	connectMock.On("SendSSHPublicKey", mock.Anything, mock.Anything).Return(
		&ec2instanceconnect.SendSSHPublicKeyOutput{Success: true}, nil,
	)

	newEC2Client = func(cfg aws.Config, logger *log.Logger) (*ec2client.Client, error) {
		return ec2client.NewTestClient(ec2Mock, connectMock, new(mockHTTPRequestSigner)), nil
	}

	// Mock getPublicKey (used when identity file is provided)
	getPublicKey = func(path string) (string, error) {
		assert.Equal(t, "/home/user/.ssh/id_rsa", path)
		return "ssh-rsa AAAAB3Nza... user@host", nil
	}

	var captured commandCapture
	executeCommand = func(cmd string, args []string, logger *log.Logger) error {
		captured.command = cmd
		captured.args = args
		return nil
	}

	session, err := NewSSHSession([]string{"-i", "/home/user/.ssh/id_rsa", "i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	// Verify identity file is used
	assert.Contains(t, captured.args, "-i/home/user/.ssh/id_rsa")
}

func TestSSHSession_Run_WithCommand(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSSHSession([]string{"i-1234567890abcdef0", "--", "ls", "-la"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	// Command should be at the end after --
	assert.Contains(t, captured.args, "--")
	assert.Contains(t, captured.args, "ls")
	assert.Contains(t, captured.args, "-la")
}

// =============================================================================
// Error Propagation Tests
// =============================================================================

func TestSSHSession_Run_AWSConfigError(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	origLoadAWSConfig := loadAWSConfig
	t.Cleanup(func() { loadAWSConfig = origLoadAWSConfig })

	expectedErr := errors.New("no credentials found")
	loadAWSConfig = func(region, profile string, logger *log.Logger) (aws.Config, error) {
		return aws.Config{}, expectedErr
	}

	session, err := NewSSHSession([]string{"i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestSSHSession_Run_EC2ClientError(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	origLoadAWSConfig := loadAWSConfig
	origNewEC2Client := newEC2Client
	t.Cleanup(func() {
		loadAWSConfig = origLoadAWSConfig
		newEC2Client = origNewEC2Client
	})

	loadAWSConfig = func(region, profile string, logger *log.Logger) (aws.Config, error) {
		return aws.Config{Region: "us-east-1"}, nil
	}

	expectedErr := errors.New("failed to create client")
	newEC2Client = func(cfg aws.Config, logger *log.Logger) (*ec2client.Client, error) {
		return nil, expectedErr
	}

	session, err := NewSSHSession([]string{"i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestSSHSession_Run_InstanceNotFound(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	origLoadAWSConfig := loadAWSConfig
	origNewEC2Client := newEC2Client
	t.Cleanup(func() {
		loadAWSConfig = origLoadAWSConfig
		newEC2Client = origNewEC2Client
	})

	loadAWSConfig = func(region, profile string, logger *log.Logger) (aws.Config, error) {
		return aws.Config{Region: "us-east-1"}, nil
	}

	ec2Mock := new(mockEC2API)
	// Return empty result
	ec2Mock.On("DescribeInstances", mock.Anything, mock.Anything).Return(
		&ec2.DescribeInstancesOutput{Reservations: []types.Reservation{}}, nil,
	)

	newEC2Client = func(cfg aws.Config, logger *log.Logger) (*ec2client.Client, error) {
		return ec2client.NewTestClient(ec2Mock, new(mockEC2InstanceConnectAPI), new(mockHTTPRequestSigner)), nil
	}

	session, err := NewSSHSession([]string{"i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to get instance")
}

func TestSSHSession_Run_KeygenError(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	origLoadAWSConfig := loadAWSConfig
	origNewEC2Client := newEC2Client
	origGenerateKeypair := generateKeypair
	t.Cleanup(func() {
		loadAWSConfig = origLoadAWSConfig
		newEC2Client = origNewEC2Client
		generateKeypair = origGenerateKeypair
	})

	loadAWSConfig = func(region, profile string, logger *log.Logger) (aws.Config, error) {
		return aws.Config{Region: "us-east-1"}, nil
	}

	ec2Mock := new(mockEC2API)
	ec2Mock.On("DescribeInstances", mock.Anything, mock.Anything).Return(
		&ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{{Instances: []types.Instance{testInstance}}},
		}, nil,
	)

	newEC2Client = func(cfg aws.Config, logger *log.Logger) (*ec2client.Client, error) {
		return ec2client.NewTestClient(ec2Mock, new(mockEC2InstanceConnectAPI), new(mockHTTPRequestSigner)), nil
	}

	expectedErr := errors.New("ssh-keygen not found")
	generateKeypair = func(tmpDir string) (string, string, error) {
		return "", "", expectedErr
	}

	session, err := NewSSHSession([]string{"i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to generate ephemeral SSH keypair")
}

func TestSSHSession_Run_SendKeysError(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	origLoadAWSConfig := loadAWSConfig
	origNewEC2Client := newEC2Client
	origGenerateKeypair := generateKeypair
	t.Cleanup(func() {
		loadAWSConfig = origLoadAWSConfig
		newEC2Client = origNewEC2Client
		generateKeypair = origGenerateKeypair
	})

	loadAWSConfig = func(region, profile string, logger *log.Logger) (aws.Config, error) {
		return aws.Config{Region: "us-east-1"}, nil
	}

	ec2Mock := new(mockEC2API)
	connectMock := new(mockEC2InstanceConnectAPI)

	ec2Mock.On("DescribeInstances", mock.Anything, mock.Anything).Return(
		&ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{{Instances: []types.Instance{testInstance}}},
		}, nil,
	)

	// SendSSHPublicKey returns error
	expectedErr := errors.New("access denied")
	connectMock.On("SendSSHPublicKey", mock.Anything, mock.Anything).Return(
		nil, expectedErr,
	)

	newEC2Client = func(cfg aws.Config, logger *log.Logger) (*ec2client.Client, error) {
		return ec2client.NewTestClient(ec2Mock, connectMock, new(mockHTTPRequestSigner)), nil
	}

	generateKeypair = func(tmpDir string) (string, string, error) {
		return "/tmp/key", "ssh-ed25519 AAAA...", nil
	}

	session, err := NewSSHSession([]string{"i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to send SSH public key")
}

func TestSSHSession_Run_CommandError(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	origLoadAWSConfig := loadAWSConfig
	origNewEC2Client := newEC2Client
	origGenerateKeypair := generateKeypair
	origExecuteCommand := executeCommand
	t.Cleanup(func() {
		loadAWSConfig = origLoadAWSConfig
		newEC2Client = origNewEC2Client
		generateKeypair = origGenerateKeypair
		executeCommand = origExecuteCommand
	})

	loadAWSConfig = func(region, profile string, logger *log.Logger) (aws.Config, error) {
		return aws.Config{Region: "us-east-1"}, nil
	}

	ec2Mock := new(mockEC2API)
	connectMock := new(mockEC2InstanceConnectAPI)

	ec2Mock.On("DescribeInstances", mock.Anything, mock.Anything).Return(
		&ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{{Instances: []types.Instance{testInstance}}},
		}, nil,
	)
	connectMock.On("SendSSHPublicKey", mock.Anything, mock.Anything).Return(
		&ec2instanceconnect.SendSSHPublicKeyOutput{Success: true}, nil,
	)

	newEC2Client = func(cfg aws.Config, logger *log.Logger) (*ec2client.Client, error) {
		return ec2client.NewTestClient(ec2Mock, connectMock, new(mockHTTPRequestSigner)), nil
	}

	generateKeypair = func(tmpDir string) (string, string, error) {
		return "/tmp/key", "ssh-ed25519 AAAA...", nil
	}

	expectedErr := errors.New("connection refused")
	executeCommand = func(cmd string, args []string, logger *log.Logger) error {
		return expectedErr
	}

	session, err := NewSSHSession([]string{"i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

// =============================================================================
// SCPSession.Run() Integration Tests
// =============================================================================

func TestSCPSession_Run_Upload(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSCPSession([]string{"/local/file.txt", "i-1234567890abcdef0:/remote/file.txt"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	assert.Equal(t, "scp", captured.command)
	// Upload: local first, then remote
	assert.Equal(t, "/local/file.txt", captured.args[len(captured.args)-2])
	// Remote path includes resolved address
	lastArg := captured.args[len(captured.args)-1]
	assert.Contains(t, lastArg, "10.0.0.1:/remote/file.txt")
}

func TestSCPSession_Run_Download(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSCPSession([]string{"i-1234567890abcdef0:/remote/file.txt", "/local/"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	assert.Equal(t, "scp", captured.command)
	// Download: remote first, then local
	assert.Contains(t, captured.args[len(captured.args)-2], "10.0.0.1:/remote/file.txt")
	assert.Equal(t, "/local/", captured.args[len(captured.args)-1])
}

func TestSCPSession_Run_WithPort(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSCPSession([]string{"-P", "2222", "/local/file.txt", "i-1234567890abcdef0:/remote/"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	// Port flag is passthrough: appears as separate args "-P" and "2222"
	assert.Contains(t, captured.args, "-P")
	assert.Contains(t, captured.args, "2222")
}

// =============================================================================
// SFTPSession.Run() Integration Tests
// =============================================================================

func TestSFTPSession_Run_Success(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSFTPSession([]string{"ec2-user@i-1234567890abcdef0:/home/ec2-user"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	assert.Equal(t, "sftp", captured.command)
	// SFTP destination format: user@host:path
	lastArg := captured.args[len(captured.args)-1]
	assert.Contains(t, lastArg, "ec2-user@10.0.0.1:/home/ec2-user")
}

func TestSFTPSession_Run_WithPort(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSFTPSession([]string{"-P", "2222", "i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	// Port flag is passthrough: appears as separate args "-P" and "2222"
	assert.Contains(t, captured.args, "-P")
	assert.Contains(t, captured.args, "2222")
}

// =============================================================================
// Address Type Tests
// =============================================================================

func TestSSHSession_Run_WithPublicAddress(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSSHSession([]string{"--address-type", "public", "i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	// Should use public IP
	assert.Equal(t, "52.1.2.3", captured.args[len(captured.args)-1])
}

func TestSSHSession_Run_WithPrivateAddress(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSSHSession([]string{"--address-type", "private", "i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	// Should use private IP
	assert.Equal(t, "10.0.0.1", captured.args[len(captured.args)-1])
}

// =============================================================================
// Edge Case Tests - getPublicKey Error and setupProxyCommand Branches
// =============================================================================

func TestSSHSession_Run_GetPublicKeyError(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	// Save originals
	origLoadAWSConfig := loadAWSConfig
	origNewEC2Client := newEC2Client
	origGetPublicKey := getPublicKey
	origExecuteCommand := executeCommand

	t.Cleanup(func() {
		loadAWSConfig = origLoadAWSConfig
		newEC2Client = origNewEC2Client
		getPublicKey = origGetPublicKey
		executeCommand = origExecuteCommand
	})

	// Mock AWS config
	loadAWSConfig = func(region, profile string, logger *log.Logger) (aws.Config, error) {
		return aws.Config{Region: "us-east-1"}, nil
	}

	// Mock EC2 client
	ec2Mock := new(mockEC2API)
	connectMock := new(mockEC2InstanceConnectAPI)

	ec2Mock.On("DescribeInstances", mock.Anything, mock.Anything).Return(
		&ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{{Instances: []types.Instance{testInstance}}},
		}, nil,
	)
	connectMock.On("SendSSHPublicKey", mock.Anything, mock.Anything).Return(
		&ec2instanceconnect.SendSSHPublicKeyOutput{Success: true}, nil,
	)

	newEC2Client = func(cfg aws.Config, logger *log.Logger) (*ec2client.Client, error) {
		return ec2client.NewTestClient(ec2Mock, connectMock, new(mockHTTPRequestSigner)), nil
	}

	// Mock getPublicKey to return an error
	expectedErr := errors.New("permission denied reading key file")
	getPublicKey = func(path string) (string, error) {
		return "", expectedErr
	}

	executeCommand = func(cmd string, args []string, logger *log.Logger) error {
		return nil
	}

	// Use -i to trigger the getPublicKey path instead of generateKeypair
	session, err := NewSSHSession([]string{"-i", "/home/user/.ssh/id_rsa", "i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to read public key from /home/user/.ssh/id_rsa")
}

func TestSSHSession_Run_WithExplicitEICEID(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	// This test verifies that when an explicit EICE ID is provided,
	// GuessEICEByVPCAndSubnet is NOT called (the explicit ID is used directly)

	var captured commandCapture
	ec2Mock, _ := setupMocksForRun(t, testInstance, &captured)

	// Note: We do NOT set up a mock for DescribeInstanceConnectEndpoints
	// because it should never be called when an explicit EICE ID is provided

	session, err := NewSSHSession([]string{"--eice-id", "eice-explicit123", "i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	// Verify ProxyCommand contains the explicit EICE ID
	foundProxy := false
	for _, arg := range captured.args {
		if len(arg) > 15 && arg[:15] == "-oProxyCommand=" {
			assert.Contains(t, arg, "--eice-tunnel")
			assert.Contains(t, arg, "--eice-id eice-explicit123")
			foundProxy = true
		}
	}
	assert.True(t, foundProxy, "ProxyCommand should be set with explicit EICE ID")

	// Verify GuessEICEByVPCAndSubnet was NOT called
	ec2Mock.AssertNotCalled(t, "DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything)
}

func TestSSHSession_Run_EICEAutoDiscovery(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	// This test verifies EICE auto-discovery when --use-eice is provided
	// WITHOUT an explicit --eice-id. The code should call GuessEICEByVPCAndSubnet
	// which uses DescribeInstanceConnectEndpoints to find an EICE endpoint.

	// Save originals
	origLoadAWSConfig := loadAWSConfig
	origNewEC2Client := newEC2Client
	origGenerateKeypair := generateKeypair
	origExecuteCommand := executeCommand

	t.Cleanup(func() {
		loadAWSConfig = origLoadAWSConfig
		newEC2Client = origNewEC2Client
		generateKeypair = origGenerateKeypair
		executeCommand = origExecuteCommand
	})

	loadAWSConfig = func(region, profile string, logger *log.Logger) (aws.Config, error) {
		return aws.Config{Region: "us-east-1"}, nil
	}

	ec2Mock := new(mockEC2API)
	connectMock := new(mockEC2InstanceConnectAPI)

	// Mock DescribeInstances to return test instance
	ec2Mock.On("DescribeInstances", mock.Anything, mock.Anything).Return(
		&ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{{Instances: []types.Instance{testInstance}}},
		}, nil,
	)

	// Mock DescribeInstanceConnectEndpoints for EICE auto-discovery
	// The paginator will call this to find EICE endpoints in the VPC
	ec2Mock.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
		&ec2.DescribeInstanceConnectEndpointsOutput{
			InstanceConnectEndpoints: []types.Ec2InstanceConnectEndpoint{
				{
					InstanceConnectEndpointId: aws.String("eice-autodiscovered"),
					VpcId:                     aws.String("vpc-123"),
					SubnetId:                  aws.String("subnet-456"),
					DnsName:                   aws.String("eice.us-east-1.amazonaws.com"),
				},
			},
		}, nil,
	)

	connectMock.On("SendSSHPublicKey", mock.Anything, mock.Anything).Return(
		&ec2instanceconnect.SendSSHPublicKeyOutput{Success: true}, nil,
	)

	newEC2Client = func(cfg aws.Config, logger *log.Logger) (*ec2client.Client, error) {
		return ec2client.NewTestClient(ec2Mock, connectMock, new(mockHTTPRequestSigner)), nil
	}

	generateKeypair = func(tmpDir string) (string, string, error) {
		return "/tmp/test_key", "ssh-ed25519 AAAAC3NzaC1... test@host", nil
	}

	var captured commandCapture
	executeCommand = func(cmd string, args []string, logger *log.Logger) error {
		captured.command = cmd
		captured.args = args
		return nil
	}

	// Use --use-eice WITHOUT --eice-id to trigger auto-discovery
	session, err := NewSSHSession([]string{"--use-eice", "i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	// Verify ProxyCommand contains the auto-discovered EICE ID
	foundProxy := false
	for _, arg := range captured.args {
		if len(arg) > 15 && arg[:15] == "-oProxyCommand=" {
			assert.Contains(t, arg, "--eice-tunnel")
			assert.Contains(t, arg, "--eice-id eice-autodiscovered")
			foundProxy = true
		}
	}
	assert.True(t, foundProxy, "ProxyCommand should contain auto-discovered EICE ID")

	// Verify DescribeInstanceConnectEndpoints WAS called for auto-discovery
	ec2Mock.AssertCalled(t, "DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything)
}

func TestSSHSession_Run_EICEAutoDiscoveryError(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	// This test verifies error handling when EICE auto-discovery fails
	// (e.g., no EICE endpoint found in the VPC)

	// Save originals
	origLoadAWSConfig := loadAWSConfig
	origNewEC2Client := newEC2Client
	origGenerateKeypair := generateKeypair
	origExecuteCommand := executeCommand

	t.Cleanup(func() {
		loadAWSConfig = origLoadAWSConfig
		newEC2Client = origNewEC2Client
		generateKeypair = origGenerateKeypair
		executeCommand = origExecuteCommand
	})

	loadAWSConfig = func(region, profile string, logger *log.Logger) (aws.Config, error) {
		return aws.Config{Region: "us-east-1"}, nil
	}

	ec2Mock := new(mockEC2API)
	connectMock := new(mockEC2InstanceConnectAPI)

	// Mock DescribeInstances to return test instance
	ec2Mock.On("DescribeInstances", mock.Anything, mock.Anything).Return(
		&ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{{Instances: []types.Instance{testInstance}}},
		}, nil,
	)

	// Mock DescribeInstanceConnectEndpoints to return EMPTY results (no EICE found)
	ec2Mock.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
		&ec2.DescribeInstanceConnectEndpointsOutput{
			InstanceConnectEndpoints: []types.Ec2InstanceConnectEndpoint{},
		}, nil,
	)

	connectMock.On("SendSSHPublicKey", mock.Anything, mock.Anything).Return(
		&ec2instanceconnect.SendSSHPublicKeyOutput{Success: true}, nil,
	)

	newEC2Client = func(cfg aws.Config, logger *log.Logger) (*ec2client.Client, error) {
		return ec2client.NewTestClient(ec2Mock, connectMock, new(mockHTTPRequestSigner)), nil
	}

	generateKeypair = func(tmpDir string) (string, string, error) {
		return "/tmp/test_key", "ssh-ed25519 AAAAC3NzaC1... test@host", nil
	}

	executeCommand = func(cmd string, args []string, logger *log.Logger) error {
		return nil
	}

	// Use --use-eice WITHOUT --eice-id - should fail because no EICE found
	session, err := NewSSHSession([]string{"--use-eice", "i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to find EICE endpoint")
}

func TestSSHSession_Run_ProxyCommandWithFlags(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	// This test verifies that Region, Profile, and Debug flags
	// are correctly passed to the ProxyCommand

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	// Use SSM with region, profile, and debug flags
	session, err := NewSSHSession([]string{
		"--use-ssm",
		"--region", "us-west-2",
		"--profile", "myprofile",
		"--debug",
		"i-1234567890abcdef0",
	})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	// Find and verify the ProxyCommand
	foundProxy := false
	for _, arg := range captured.args {
		if len(arg) > 15 && arg[:15] == "-oProxyCommand=" {
			foundProxy = true
			// Verify all flags are present in ProxyCommand
			assert.Contains(t, arg, "--ssm-tunnel", "ProxyCommand should contain --ssm-tunnel")
			assert.Contains(t, arg, "--region us-west-2", "ProxyCommand should contain --region flag")
			assert.Contains(t, arg, "--profile myprofile", "ProxyCommand should contain --profile flag")
			assert.Contains(t, arg, "--debug", "ProxyCommand should contain --debug flag")
		}
	}
	assert.True(t, foundProxy, "ProxyCommand should be set for SSM")
}

// =============================================================================
// Quick Win Edge Case Tests
// =============================================================================

func TestSSHSession_Run_WithDebugFlag(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	// Test that --debug flag is recognized and processed correctly
	// The initLogger() function (ssh_session.go:143-145) sets output to stderr when Debug=true

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSSHSession([]string{"--debug", "i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	// Verify ssh command was called
	assert.Equal(t, "ssh", captured.command)
	// Destination should be resolved
	assert.Equal(t, "10.0.0.1", captured.args[len(captured.args)-1])
}

func TestSCPSession_Run_PathWithSpaces(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	// Test that paths with spaces are handled correctly

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	// Upload a file with spaces in the path
	session, err := NewSCPSession([]string{"/local/path with spaces/file.txt", "i-1234567890abcdef0:/remote/dest/"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	assert.Equal(t, "scp", captured.command)
	// Verify the local path with spaces is passed correctly (not escaped, scp handles it)
	assert.Equal(t, "/local/path with spaces/file.txt", captured.args[len(captured.args)-2])
}

func TestSFTPSession_Run_PathWithSpaces(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	// Test that SFTP destination with spaces is handled correctly

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSFTPSession([]string{"i-1234567890abcdef0:/path with spaces/"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	assert.Equal(t, "sftp", captured.command)
	// Verify path with spaces is preserved
	lastArg := captured.args[len(captured.args)-1]
	assert.Contains(t, lastArg, "/path with spaces/")
}

func TestSSHSession_Run_InteractiveShell(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	// Test SSH without a remote command (interactive shell mode)
	// Should NOT have -- separator in args

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSSHSession([]string{"i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	assert.Equal(t, "ssh", captured.command)
	// Verify no -- separator in args (interactive mode)
	for _, arg := range captured.args {
		assert.NotEqual(t, "--", arg, "interactive shell should not have -- separator")
	}
}

func TestSSHSession_Run_WithRemoteCommand(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	// Test SSH with a remote command that has spaces
	// Should have -- separator followed by command

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSSHSession([]string{"i-1234567890abcdef0", "--", "ls", "-la", "/var/log"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	assert.Equal(t, "ssh", captured.command)
	// Find the -- separator and verify command follows
	foundSep := false
	for i, arg := range captured.args {
		if arg == "--" {
			foundSep = true
			// Verify command follows
			assert.Equal(t, "ls", captured.args[i+1])
			assert.Equal(t, "-la", captured.args[i+2])
			assert.Equal(t, "/var/log", captured.args[i+3])
			break
		}
	}
	assert.True(t, foundSep, "should have -- separator for remote command")
}

func TestSSHSession_Run_UserAtDestination(t *testing.T) {
	// No t.Parallel() - modifies global DI vars

	// Test SSH with user@instance format (alternative to -l flag)

	var captured commandCapture
	setupMocksForRun(t, testInstance, &captured)

	session, err := NewSSHSession([]string{"ubuntu@i-1234567890abcdef0"})
	require.NoError(t, err)

	err = session.Run()
	require.NoError(t, err)

	assert.Equal(t, "ssh", captured.command)
	// With new target design, login is embedded in target string (not -l flag)
	// Format is preserved: user@host stays as user@resolved_host
	assert.Equal(t, "ubuntu@10.0.0.1", captured.args[len(captured.args)-1])
}
