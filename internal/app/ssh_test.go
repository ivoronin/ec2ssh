package app

import (
	"errors"
	"testing"

	"github.com/ivoronin/ec2ssh/internal/ec2client"
	"github.com/ivoronin/ec2ssh/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSSHSession(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args           []string
		wantLogin      string
		wantHost       string
		wantDstType    ec2client.DstType
		wantAddrType   ec2client.AddrType
		wantUseEICE    bool
		wantUseSSM     bool
		wantNoSendKeys bool
		wantErr        bool
		errContains    string
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
		},

		// Flags
		"login flag": {
			args:     []string{"-l", "ubuntu", "myhost"},
			wantHost: "myhost",
			// Note: -l flag sets Login, NOT target.Login()
		},
		"target login overrides flag": {
			args:      []string{"-l", "admin", "user@myhost"},
			wantLogin: "user", // target login is used, -l flag is separate passthrough
			wantHost:  "myhost",
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
				assert.True(t, errors.Is(err, ErrUsage), "expected ErrUsage, got: %v", err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, session)
			require.NotNil(t, session.Target, "target should be set")

			assert.Equal(t, tc.wantHost, session.Target.Host(), "host")
			if tc.wantLogin != "" {
				assert.Equal(t, tc.wantLogin, session.Target.Login(), "login")
			}
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
	assert.Equal(t, "myhost", session.Target.Host())
	assert.Equal(t, []string{"ls", "-la"}, session.CommandWithArgs)
}

func TestNewSSHSession_SimpleCommand(t *testing.T) {
	t.Parallel()

	// Command without flags doesn't need --
	session, err := NewSSHSession([]string{"myhost", "hostname"})
	require.NoError(t, err)
	assert.Equal(t, "myhost", session.Target.Host())
	assert.Equal(t, []string{"hostname"}, session.CommandWithArgs)
}

func TestSSHSession_BuildArgs(t *testing.T) {
	t.Parallel()

	t.Run("login from flag", func(t *testing.T) {
		t.Parallel()

		// Simulates: ec2ssh -l ec2-user myhost
		session := &SSHSession{}
		session.Target, _ = ssh.NewSSHTarget("myhost")
		session.Login = "ec2-user" // From -l flag
		session.Target.SetHost("10.0.0.1")
		session.privateKeyPath = "/tmp/key"
		session.instance.InstanceId = strPtr("i-123")
		session.PassArgs = []string{"-v"}

		args := session.buildArgs()

		// Should contain login (from flag)
		assert.Contains(t, args, "-lec2-user")
		// Should contain identity file
		assert.Contains(t, args, "-i/tmp/key")
		// Should contain passthrough args
		assert.Contains(t, args, "-v")
		// Should contain host key alias
		assert.Contains(t, args, "-oHostKeyAlias=i-123")
		// Last arg should be resolved destination
		assert.Equal(t, "10.0.0.1", args[len(args)-1])
	})

	t.Run("login embedded in target url", func(t *testing.T) {
		t.Parallel()

		// Simulates: ec2ssh ssh://ec2-user@myhost:2222
		session := &SSHSession{}
		session.Target, _ = ssh.NewSSHTarget("ssh://ec2-user@myhost:2222")
		session.Target.SetHost("10.0.0.1")
		session.privateKeyPath = "/tmp/key"
		session.instance.InstanceId = strPtr("i-123")

		args := session.buildArgs()

		// Should NOT contain -l (login embedded in target)
		for _, arg := range args {
			assert.NotContains(t, arg, "-lec2-user")
		}
		// Last arg should be full URL with resolved host (port preserved)
		assert.Equal(t, "ssh://ec2-user@10.0.0.1:2222", args[len(args)-1])
	})

	t.Run("non-URL target with login", func(t *testing.T) {
		t.Parallel()

		// Simulates: ec2ssh ec2-user@myhost
		session := &SSHSession{}
		session.Target, _ = ssh.NewSSHTarget("ec2-user@myhost")
		session.Target.SetHost("10.0.0.1")
		session.privateKeyPath = "/tmp/key"
		session.instance.InstanceId = strPtr("i-123")

		args := session.buildArgs()

		// Should NOT contain -l (login embedded in target)
		for _, arg := range args {
			assert.NotContains(t, arg, "-l")
		}
		// Last arg should be user@resolved_host
		assert.Equal(t, "ec2-user@10.0.0.1", args[len(args)-1])
	})
}

func TestSSHSession_BuildArgsWithCommand(t *testing.T) {
	t.Parallel()

	session := &SSHSession{}
	session.Target, _ = ssh.NewSSHTarget("ec2-user@myhost")
	session.Target.SetHost("10.0.0.1")
	session.privateKeyPath = "/tmp/key"
	session.instance.InstanceId = strPtr("i-123")
	session.CommandWithArgs = []string{"ls", "-la"}

	args := session.buildArgs()

	// buildArgs adds -- separator before command
	// Format: [...options, destination, --, ls, -la]
	assert.Equal(t, "ec2-user@10.0.0.1", args[len(args)-4])
	assert.Equal(t, "--", args[len(args)-3])
	assert.Equal(t, "ls", args[len(args)-2])
	assert.Equal(t, "-la", args[len(args)-1])
}

func TestSSHSession_BuildArgsWithProxyCommand(t *testing.T) {
	t.Parallel()

	session := &SSHSession{}
	session.Target, _ = ssh.NewSSHTarget("ec2-user@i-123")
	session.Target.SetHost("i-123")
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

// Passthrough mode tests - when Target is nil (e.g., ec2ssh -V)
func TestNewSSHSession_PassthroughMode(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args         []string
		wantPassArgs []string
	}{
		"version flag": {
			args:         []string{"-V"},
			wantPassArgs: []string{"-V"},
		},
		"verbose and version": {
			args:         []string{"-v", "-V"},
			wantPassArgs: []string{"-v", "-V"},
		},
		"help long form": {
			args:         []string{"--help"},
			wantPassArgs: []string{"--help"},
		},
		"option only": {
			args:         []string{"-o", "StrictHostKeyChecking=no"},
			wantPassArgs: []string{"-o", "StrictHostKeyChecking=no"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			session, err := NewSSHSession(tc.args)
			require.NoError(t, err)
			require.NotNil(t, session)
			assert.Nil(t, session.Target, "Target should be nil in passthrough mode")
			assert.Equal(t, tc.wantPassArgs, session.PassArgs)
		})
	}
}

func TestSSHSession_BuildArgs_PassthroughMode(t *testing.T) {
	t.Parallel()

	// Simulates: ec2ssh -V (passthrough to ssh -V)
	session := &SSHSession{}
	session.Target = nil // Passthrough mode
	session.PassArgs = []string{"-V"}

	args := session.buildArgs()

	// Should contain only passthrough args, no destination
	assert.Equal(t, []string{"-V"}, args)
}

// Helper to create string pointer
func strPtr(s string) *string {
	return &s
}
