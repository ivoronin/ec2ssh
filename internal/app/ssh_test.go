package app

import (
	"testing"

	"github.com/ivoronin/ec2ssh/internal/ec2client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSSHSession(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args            []string
		wantLogin       string
		wantHost        string
		wantPort        string
		wantDstType     ec2client.DstType
		wantAddrType    ec2client.AddrType
		wantUseEICE     bool
		wantUseSSM      bool
		wantNoSendKeys  bool
		wantErr         bool
		errContains     string
	}{
		// Basic destination formats
		"simple host": {
			args:     []string{"i-1234567890abcdef0"},
			wantHost: "i-1234567890abcdef0",
		},
		"user@host": {
			args:      []string{"ec2-user@i-1234567890abcdef0"},
			wantLogin: "ec2-user",
			wantHost:  "i-1234567890abcdef0",
		},
		"ssh url format": {
			args:      []string{"ssh://admin@myhost:2222"},
			wantLogin: "admin",
			wantHost:  "myhost",
			wantPort:  "2222",
		},

		// Flags
		"login flag": {
			args:      []string{"-l", "ubuntu", "myhost"},
			wantLogin: "ubuntu",
			wantHost:  "myhost",
		},
		"port flag": {
			args:     []string{"-p", "2222", "myhost"},
			wantHost: "myhost",
			wantPort: "2222",
		},
		"port flag overrides url": {
			args:     []string{"-p", "3333", "ssh://admin@myhost:2222"},
			wantHost: "myhost",
			wantPort: "3333",
		},
		"region flag": {
			args:     []string{"--region", "us-west-2", "myhost"},
			wantHost: "myhost",
		},
		"profile flag": {
			args:     []string{"--profile", "myprofile", "myhost"},
			wantHost: "myhost",
		},

		// Destination and address type flags
		"destination type id": {
			args:        []string{"--destination-type", "id", "i-123"},
			wantHost:    "i-123",
			wantDstType: ec2client.DstTypeID,
		},
		"destination type private_ip": {
			args:        []string{"--destination-type", "private_ip", "10.0.0.1"},
			wantHost:    "10.0.0.1",
			wantDstType: ec2client.DstTypePrivateIP,
		},
		"destination type name_tag": {
			args:        []string{"--destination-type", "name_tag", "my-server"},
			wantHost:    "my-server",
			wantDstType: ec2client.DstTypeNameTag,
		},
		"address type private": {
			args:         []string{"--address-type", "private", "myhost"},
			wantHost:     "myhost",
			wantAddrType: ec2client.AddrTypePrivate,
		},
		"address type public": {
			args:         []string{"--address-type", "public", "myhost"},
			wantHost:     "myhost",
			wantAddrType: ec2client.AddrTypePublic,
		},

		// Tunnel options
		"use eice flag": {
			args:        []string{"--use-eice", "myhost"},
			wantHost:    "myhost",
			wantUseEICE: true,
		},
		"use ssm flag": {
			args:       []string{"--use-ssm", "myhost"},
			wantHost:   "myhost",
			wantUseSSM: true,
		},
		"eice id implies use eice": {
			args:        []string{"--eice-id", "eice-123", "myhost"},
			wantHost:    "myhost",
			wantUseEICE: true,
		},
		"no send keys flag": {
			args:           []string{"--no-send-keys", "myhost"},
			wantHost:       "myhost",
			wantNoSendKeys: true,
		},

		// Error cases
		"missing destination": {
			args:        []string{},
			wantErr:     true,
			errContains: "missing destination",
		},
		"invalid destination type": {
			args:        []string{"--destination-type", "invalid", "myhost"},
			wantErr:     true,
			errContains: "unknown destination type",
		},
		"invalid address type": {
			args:        []string{"--address-type", "invalid", "myhost"},
			wantErr:     true,
			errContains: "unknown address type",
		},
		"eice and ssm mutually exclusive": {
			args:        []string{"--use-eice", "--use-ssm", "myhost"},
			wantErr:     true,
			errContains: "mutually exclusive",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			session, err := NewSSHSession(tc.args)

			if tc.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrUsage)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, session)

			assert.Equal(t, tc.wantHost, session.Destination, "destination")
			if tc.wantLogin != "" {
				assert.Equal(t, tc.wantLogin, session.Login, "login")
			}
			assert.Equal(t, tc.wantPort, session.Port, "port")
			assert.Equal(t, tc.wantDstType, session.DstType, "dstType")
			assert.Equal(t, tc.wantAddrType, session.AddrType, "addrType")
			assert.Equal(t, tc.wantUseEICE, session.UseEICE, "useEICE")
			assert.Equal(t, tc.wantUseSSM, session.UseSSM, "useSSM")
			assert.Equal(t, tc.wantNoSendKeys, session.NoSendKeys, "noSendKeys")
		})
	}
}

func TestNewSSHSession_PassthroughArgs(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args         []string
		wantPassArgs []string
	}{
		"ssh option passthrough": {
			args:         []string{"-o", "StrictHostKeyChecking=no", "myhost"},
			wantPassArgs: []string{"-o", "StrictHostKeyChecking=no"},
		},
		"port forwarding passthrough": {
			args:         []string{"-L", "8080:localhost:80", "myhost"},
			wantPassArgs: []string{"-L", "8080:localhost:80"},
		},
		"multiple passthrough": {
			args:         []string{"-o", "opt1", "-L", "8080:localhost:80", "-o", "opt2", "myhost"},
			wantPassArgs: []string{"-o", "opt1", "-L", "8080:localhost:80", "-o", "opt2"},
		},
		"unknown short flags passthrough": {
			args:         []string{"-X", "-Y", "myhost"},
			wantPassArgs: []string{"-X", "-Y"},
		},
		"verbose flags passthrough": {
			args:         []string{"-v", "-v", "-v", "myhost"},
			wantPassArgs: []string{"-v", "-v", "-v"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			session, err := NewSSHSession(tc.args)
			require.NoError(t, err)
			assert.Equal(t, tc.wantPassArgs, session.PassArgs)
		})
	}
}

func TestNewSSHSession_CommandAfterDestination(t *testing.T) {
	t.Parallel()

	// Note: -la would be parsed as -l (login) with value "a"
	// To pass flags to remote command, use -- separator
	session, err := NewSSHSession([]string{"myhost", "--", "ls", "-la"})
	require.NoError(t, err)
	assert.Equal(t, "myhost", session.Destination)
	assert.Equal(t, []string{"ls", "-la"}, session.CommandWithArgs)
}

func TestNewSSHSession_SimpleCommand(t *testing.T) {
	t.Parallel()

	// Command without flags doesn't need --
	session, err := NewSSHSession([]string{"myhost", "hostname"})
	require.NoError(t, err)
	assert.Equal(t, "myhost", session.Destination)
	assert.Equal(t, []string{"hostname"}, session.CommandWithArgs)
}

func TestSSHSession_BuildArgs(t *testing.T) {
	t.Parallel()

	// Note: buildArgs requires runtime state to be set (destinationAddr, privateKeyPath, instance)
	// which is normally done in run(). For unit tests of buildArgs, we'd need to
	// set these fields manually or test through higher-level integration tests.

	// This test focuses on argument construction logic by setting up a minimal session
	session := &SSHSession{}
	session.Login = "ec2-user"
	session.Port = "2222"
	session.destinationAddr = "10.0.0.1"
	session.privateKeyPath = "/tmp/key"
	session.instance.InstanceId = strPtr("i-123")
	session.PassArgs = []string{"-v"}

	args := session.buildArgs()

	// Should contain login
	assert.Contains(t, args, "-lec2-user")
	// Should contain port
	assert.Contains(t, args, "-p2222")
	// Should contain identity file
	assert.Contains(t, args, "-i/tmp/key")
	// Should contain passthrough args
	assert.Contains(t, args, "-v")
	// Should contain host key alias
	assert.Contains(t, args, "-oHostKeyAlias=i-123")
	// Last arg should be destination
	assert.Equal(t, "10.0.0.1", args[len(args)-1])
}

func TestSSHSession_BuildArgsWithCommand(t *testing.T) {
	t.Parallel()

	session := &SSHSession{}
	session.Login = "ec2-user"
	session.destinationAddr = "10.0.0.1"
	session.privateKeyPath = "/tmp/key"
	session.instance.InstanceId = strPtr("i-123")
	session.CommandWithArgs = []string{"ls", "-la"}

	args := session.buildArgs()

	// buildArgs adds -- separator before command
	// Format: [...options, destination, --, ls, -la]
	assert.Equal(t, "10.0.0.1", args[len(args)-4])
	assert.Equal(t, "--", args[len(args)-3])
	assert.Equal(t, "ls", args[len(args)-2])
	assert.Equal(t, "-la", args[len(args)-1])
}

func TestSSHSession_BuildArgsWithProxyCommand(t *testing.T) {
	t.Parallel()

	session := &SSHSession{}
	session.Login = "ec2-user"
	session.destinationAddr = "i-123"
	session.privateKeyPath = "/tmp/key"
	session.instance.InstanceId = strPtr("i-123")
	session.proxyCommand = "ec2ssh --eice-tunnel --host 10.0.0.1 --port %p --eice-id eice-123"

	args := session.buildArgs()

	// Should contain ProxyCommand
	found := false
	for _, arg := range args {
		if arg == "-oProxyCommand=ec2ssh --eice-tunnel --host 10.0.0.1 --port %p --eice-id eice-123" {
			found = true
			break
		}
	}
	assert.True(t, found, "ProxyCommand should be in args")
}

// Helper to create string pointer
func strPtr(s string) *string {
	return &s
}
