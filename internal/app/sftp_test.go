package app

import (
	"testing"

	"github.com/ivoronin/ec2ssh/internal/ec2client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSFTPSession(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args           []string
		wantHost       string
		wantLogin      string
		wantPort       string
		wantRemotePath string
		wantDstType    ec2client.DstType
		wantErr        bool
		errContains    string
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
			args:           []string{"myhost:/home/user"},
			wantHost:       "myhost",
			wantRemotePath: "/home/user",
		},
		"user@host:path": {
			args:           []string{"admin@myhost:/home/user"},
			wantLogin:      "admin",
			wantHost:       "myhost",
			wantRemotePath: "/home/user",
		},

		// SFTP URL format
		"sftp url host only": {
			args:     []string{"sftp://myhost"},
			wantHost: "myhost",
		},
		"sftp url with port": {
			args:     []string{"sftp://myhost:2222"},
			wantHost: "myhost",
			wantPort: "2222",
		},
		"sftp url full": {
			args:           []string{"sftp://admin@myhost:2222/home/user"},
			wantLogin:      "admin",
			wantHost:       "myhost",
			wantPort:       "2222",
			wantRemotePath: "home/user",
		},

		// Port flag
		"port flag": {
			args:     []string{"-P", "3333", "myhost"},
			wantHost: "myhost",
			wantPort: "3333",
		},
		"port flag overrides url": {
			args:     []string{"-P", "3333", "sftp://myhost:2222"},
			wantHost: "myhost",
			wantPort: "3333",
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

		// IPv6
		"ipv6": {
			args:           []string{"[::1]:/path"},
			wantHost:       "::1",
			wantRemotePath: "/path",
		},
		"sftp url ipv6": {
			args:           []string{"sftp://[2001:db8::1]:22/data"},
			wantHost:       "2001:db8::1",
			wantPort:       "22",
			wantRemotePath: "data",
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

			assert.Equal(t, tc.wantHost, session.Destination, "destination")
			if tc.wantLogin != "" {
				assert.Equal(t, tc.wantLogin, session.Login, "login")
			}
			assert.Equal(t, tc.wantPort, session.Port, "port")
			assert.Equal(t, tc.wantRemotePath, session.RemotePath, "remotePath")
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
		session.Login = "ec2-user"
		session.Port = "2222"
		session.RemotePath = "/home/ec2-user"
		session.destinationAddr = "10.0.0.1"
		session.privateKeyPath = "/tmp/key"
		session.instance.InstanceId = strPtr("i-123")

		args := session.buildArgs()

		// Should contain port (uppercase -P for SFTP)
		assert.Contains(t, args, "-P2222")
		// Should contain identity file
		assert.Contains(t, args, "-i/tmp/key")
		// Last arg: login@host:path
		assert.Equal(t, "ec2-user@10.0.0.1:/home/ec2-user", args[len(args)-1])
	})

	t.Run("without path", func(t *testing.T) {
		t.Parallel()

		session := &SFTPSession{}
		session.Login = "admin"
		session.RemotePath = ""
		session.destinationAddr = "10.0.0.1"
		session.privateKeyPath = "/tmp/key"
		session.instance.InstanceId = strPtr("i-123")

		args := session.buildArgs()

		// Last arg: login@host (no path)
		assert.Equal(t, "admin@10.0.0.1", args[len(args)-1])
	})

	t.Run("without login", func(t *testing.T) {
		t.Parallel()

		session := &SFTPSession{}
		session.Login = ""
		session.RemotePath = "/path"
		session.destinationAddr = "host"
		session.privateKeyPath = "/tmp/key"
		session.instance.InstanceId = strPtr("i-123")

		args := session.buildArgs()

		// Last arg: host:path (no login@)
		assert.Equal(t, "host:/path", args[len(args)-1])
	})
}
