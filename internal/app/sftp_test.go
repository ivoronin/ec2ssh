package app

import (
	"testing"

	"github.com/ivoronin/ec2ssh/internal/ssh"
	"github.com/ivoronin/ec2ssh/internal/ec2client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSFTPSession(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args        []string
		wantHost    string
		wantLogin   string
		wantDstType ec2client.DstType
		wantErr     bool
		errContains string
	}{
		// Basic formats
		"simple host": {
			args:     []string{"myhost"},
			wantHost: "myhost",
		},
		"user@host": {
			args:      []string{"admin@myhost"},
			wantLogin: "admin",
			wantHost:  "myhost",
		},
		"host:path": {
			args:     []string{"myhost:/home/user"},
			wantHost: "myhost",
		},
		"user@host:path": {
			args:      []string{"admin@myhost:/home/user"},
			wantLogin: "admin",
			wantHost:  "myhost",
		},

		// SFTP URL format
		"sftp url host only": {
			args:     []string{"sftp://myhost"},
			wantHost: "myhost",
		},
		"sftp url with port": {
			args:     []string{"sftp://myhost:2222"},
			wantHost: "myhost",
		},
		"sftp url full": {
			args:      []string{"sftp://admin@myhost:2222/home/user"},
			wantLogin: "admin",
			wantHost:  "myhost",
		},

		// Port flag
		"port flag": {
			args:     []string{"-P", "3333", "myhost"},
			wantHost: "myhost",
		},
		"target port overrides flag": {
			args:     []string{"-P", "3333", "sftp://myhost:2222"},
			wantHost: "myhost",
		},

		// Instance IDs
		"instance id": {
			args:     []string{"i-1234567890abcdef0"},
			wantHost: "i-1234567890abcdef0",
		},
		"user@instance id": {
			args:      []string{"ec2-user@i-1234567890abcdef0"},
			wantLogin: "ec2-user",
			wantHost:  "i-1234567890abcdef0",
		},

		// IPv6 - brackets stripped, Host() returns raw IPv6
		"ipv6": {
			args:     []string{"[::1]:/path"},
			wantHost: "::1", // Host() returns raw IPv6, String() adds brackets
		},
		"sftp url ipv6": {
			args:     []string{"sftp://[2001:db8::1]:22/data"},
			wantHost: "2001:db8::1", // Host() returns raw IPv6, String() adds brackets
		},

		// ec2ssh flags
		"with region": {
			args:     []string{"--region", "us-west-2", "myhost"},
			wantHost: "myhost",
		},
		"with destination type": {
			args:        []string{"--destination-type", "name_tag", "my-server"},
			wantHost:    "my-server",
			wantDstType: ec2client.DstTypeNameTag,
		},
		"with use eice": {
			args:     []string{"--use-eice", "myhost"},
			wantHost: "myhost",
		},

		// Error cases
		"invalid destination type": {
			args:        []string{"--destination-type", "invalid", "myhost"},
			wantErr:     true,
			errContains: "unknown destination type",
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

			session, err := NewSFTPSession(tc.args)

			if tc.wantErr {
				require.Error(t, err)
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
		})
	}
}

func TestNewSFTPSession_PassthroughArgs(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args         []string
		wantPassArgs []string
	}{
		"buffer size passthrough": {
			args:         []string{"-B", "32768", "myhost"},
			wantPassArgs: []string{"-B", "32768"},
		},
		"batch file passthrough": {
			args:         []string{"-b", "commands.txt", "myhost"},
			wantPassArgs: []string{"-b", "commands.txt"},
		},
		"cipher passthrough": {
			args:         []string{"-c", "aes256-ctr", "myhost"},
			wantPassArgs: []string{"-c", "aes256-ctr"},
		},
		"option passthrough": {
			args:         []string{"-o", "StrictHostKeyChecking=no", "myhost"},
			wantPassArgs: []string{"-o", "StrictHostKeyChecking=no"},
		},
		"multiple passthrough": {
			args:         []string{"-B", "32768", "-o", "opt1", "myhost"},
			wantPassArgs: []string{"-B", "32768", "-o", "opt1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			session, err := NewSFTPSession(tc.args)
			require.NoError(t, err)
			assert.Equal(t, tc.wantPassArgs, session.PassArgs)
		})
	}
}

func TestSFTPSession_BuildArgs(t *testing.T) {
	t.Parallel()

	t.Run("with path", func(t *testing.T) {
		t.Parallel()

		session := &SFTPSession{}
		session.Target, _ = ssh.NewSFTPTarget("ec2-user@myhost:/home/ec2-user")
		session.Target.SetHost("10.0.0.1")
		session.privateKeyPath = "/tmp/key"
		session.instance.InstanceId = strPtr("i-123")

		args := session.buildArgs()

		// Should contain identity file
		assert.Contains(t, args, "-i/tmp/key")
		// Last arg: login@host:path
		assert.Equal(t, "ec2-user@10.0.0.1:/home/ec2-user", args[len(args)-1])
	})

	t.Run("without path", func(t *testing.T) {
		t.Parallel()

		session := &SFTPSession{}
		session.Target, _ = ssh.NewSFTPTarget("admin@myhost")
		session.Target.SetHost("10.0.0.1")
		session.privateKeyPath = "/tmp/key"
		session.instance.InstanceId = strPtr("i-123")

		args := session.buildArgs()

		// Last arg: login@host (no path)
		assert.Equal(t, "admin@10.0.0.1", args[len(args)-1])
	})

	t.Run("without login", func(t *testing.T) {
		t.Parallel()

		session := &SFTPSession{}
		session.Target, _ = ssh.NewSFTPTarget("host:/path")
		session.Target.SetHost("host")
		session.privateKeyPath = "/tmp/key"
		session.instance.InstanceId = strPtr("i-123")

		args := session.buildArgs()

		// Last arg: host:path (no login@)
		assert.Equal(t, "host:/path", args[len(args)-1])
	})
}

// Passthrough mode tests - when Target is nil (e.g., ec2sftp -h)
func TestNewSFTPSession_PassthroughMode(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args         []string
		wantPassArgs []string
	}{
		"help flag": {
			args:         []string{"-h"},
			wantPassArgs: []string{"-h"},
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

			session, err := NewSFTPSession(tc.args)
			require.NoError(t, err)
			require.NotNil(t, session)
			assert.Nil(t, session.Target, "Target should be nil in passthrough mode")
			assert.Equal(t, tc.wantPassArgs, session.PassArgs)
		})
	}
}

func TestSFTPSession_BuildArgs_PassthroughMode(t *testing.T) {
	t.Parallel()

	// Simulates: ec2sftp -h (passthrough to sftp -h)
	session := &SFTPSession{}
	session.Target = nil // Passthrough mode
	session.PassArgs = []string{"-h"}

	args := session.buildArgs()

	// Should contain only passthrough args, no destination
	assert.Equal(t, []string{"-h"}, args)
}
