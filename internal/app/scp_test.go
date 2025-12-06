package app

import (
	"testing"

	"github.com/ivoronin/ec2ssh/internal/ec2client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSCPSession(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args           []string
		wantHost       string
		wantLogin      string
		wantLocalPath  string
		wantRemotePath string
		wantIsUpload   bool
		wantPort       string
		wantDstType    ec2client.DstType
		wantErr        bool
		errContains    string
	}{
		// Upload scenarios
		"upload local to remote": {
			args:           []string{"/local/file.txt", "host:/remote/file.txt"},
			wantHost:       "host",
			wantLocalPath:  "/local/file.txt",
			wantRemotePath: "/remote/file.txt",
			wantIsUpload:   true,
		},
		"upload with user": {
			args:           []string{"file.txt", "admin@host:/path"},
			wantHost:       "host",
			wantLogin:      "admin",
			wantLocalPath:  "file.txt",
			wantRemotePath: "/path",
			wantIsUpload:   true,
		},
		"upload to instance id": {
			args:           []string{"file.txt", "ec2-user@i-1234567890abcdef0:/home/ec2-user/"},
			wantHost:       "i-1234567890abcdef0",
			wantLogin:      "ec2-user",
			wantLocalPath:  "file.txt",
			wantRemotePath: "/home/ec2-user/",
			wantIsUpload:   true,
		},

		// Download scenarios
		"download remote to local": {
			args:           []string{"host:/remote/file.txt", "/local/"},
			wantHost:       "host",
			wantLocalPath:  "/local/",
			wantRemotePath: "/remote/file.txt",
			wantIsUpload:   false,
		},
		"download with user": {
			args:           []string{"admin@host:/path", "."},
			wantHost:       "host",
			wantLogin:      "admin",
			wantLocalPath:  ".",
			wantRemotePath: "/path",
			wantIsUpload:   false,
		},

		// Port flag
		"with port flag": {
			args:           []string{"-P", "2222", "file.txt", "host:/path"},
			wantHost:       "host",
			wantLocalPath:  "file.txt",
			wantRemotePath: "/path",
			wantPort:       "2222",
			wantIsUpload:   true,
		},

		// IPv6
		"upload to ipv6": {
			args:           []string{"file.txt", "[::1]:/remote"},
			wantHost:       "::1",
			wantLocalPath:  "file.txt",
			wantRemotePath: "/remote",
			wantIsUpload:   true,
		},

		// With ec2ssh flags
		"with region flag": {
			args:           []string{"--region", "us-west-2", "file.txt", "host:/path"},
			wantHost:       "host",
			wantLocalPath:  "file.txt",
			wantRemotePath: "/path",
			wantIsUpload:   true,
		},
		"with destination type": {
			args:           []string{"--destination-type", "name_tag", "file.txt", "my-server:/path"},
			wantHost:       "my-server",
			wantLocalPath:  "file.txt",
			wantRemotePath: "/path",
			wantIsUpload:   true,
			wantDstType:    ec2client.DstTypeNameTag,
		},
		"with use eice": {
			args:           []string{"--use-eice", "file.txt", "host:/path"},
			wantHost:       "host",
			wantLocalPath:  "file.txt",
			wantRemotePath: "/path",
			wantIsUpload:   true,
		},

		// Error cases
		"missing operands": {
			args:        []string{},
			wantErr:     true,
			errContains: "exactly 2 operands",
		},
		"single operand": {
			args:        []string{"host:/path"},
			wantErr:     true,
			errContains: "exactly 2 operands",
		},
		"both local": {
			args:        []string{"/local/a", "/local/b"},
			wantErr:     true,
			errContains: "no remote operand",
		},
		"both remote": {
			args:        []string{"host1:/path1", "host2:/path2"},
			wantErr:     true,
			errContains: "multiple remote operands",
		},
		"empty remote path": {
			args:        []string{"file.txt", "host:"},
			wantErr:     true,
			errContains: "remote path cannot be empty",
		},
		"invalid destination type": {
			args:        []string{"--destination-type", "invalid", "file.txt", "host:/path"},
			wantErr:     true,
			errContains: "unknown destination type",
		},
		"eice and ssm mutually exclusive": {
			args:        []string{"--use-eice", "--use-ssm", "file.txt", "host:/path"},
			wantErr:     true,
			errContains: "mutually exclusive",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			session, err := NewSCPSession(tc.args)

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
			assert.Equal(t, tc.wantLocalPath, session.LocalPath, "localPath")
			assert.Equal(t, tc.wantRemotePath, session.RemotePath, "remotePath")
			assert.Equal(t, tc.wantIsUpload, session.IsUpload, "isUpload")
			assert.Equal(t, tc.wantPort, session.Port, "port")
			assert.Equal(t, tc.wantDstType, session.DstType, "dstType")
		})
	}
}

func TestNewSCPSession_PassthroughArgs(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args         []string
		wantPassArgs []string
	}{
		"cipher passthrough": {
			args:         []string{"-c", "aes256-ctr", "file.txt", "host:/path"},
			wantPassArgs: []string{"-c", "aes256-ctr"},
		},
		"config file passthrough": {
			args:         []string{"-F", "/path/to/config", "file.txt", "host:/path"},
			wantPassArgs: []string{"-F", "/path/to/config"},
		},
		"option passthrough": {
			args:         []string{"-o", "StrictHostKeyChecking=no", "file.txt", "host:/path"},
			wantPassArgs: []string{"-o", "StrictHostKeyChecking=no"},
		},
		"recursive flag": {
			args:         []string{"-r", "dir/", "host:/path"},
			wantPassArgs: []string{"-r"},
		},
		"multiple passthrough": {
			args:         []string{"-r", "-v", "-o", "opt1", "file.txt", "host:/path"},
			wantPassArgs: []string{"-r", "-v", "-o", "opt1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			session, err := NewSCPSession(tc.args)
			require.NoError(t, err)
			assert.Equal(t, tc.wantPassArgs, session.PassArgs)
		})
	}
}

func TestSCPSession_BuildArgs(t *testing.T) {
	t.Parallel()

	t.Run("upload args", func(t *testing.T) {
		t.Parallel()

		session := &SCPSession{}
		session.Login = "ec2-user"
		session.LocalPath = "/local/file.txt"
		session.RemotePath = "/remote/file.txt"
		session.IsUpload = true
		session.Port = "2222"
		session.destinationAddr = "10.0.0.1"
		session.privateKeyPath = "/tmp/key"
		session.instance.InstanceId = strPtr("i-123")

		args := session.buildArgs()

		// Should contain port (uppercase -P for SCP)
		assert.Contains(t, args, "-P2222")
		// Should contain identity file
		assert.Contains(t, args, "-i/tmp/key")
		// Upload: local then remote
		assert.Equal(t, "/local/file.txt", args[len(args)-2])
		assert.Equal(t, "ec2-user@10.0.0.1:/remote/file.txt", args[len(args)-1])
	})

	t.Run("download args", func(t *testing.T) {
		t.Parallel()

		session := &SCPSession{}
		session.Login = "admin"
		session.LocalPath = "/local/"
		session.RemotePath = "/remote/file.txt"
		session.IsUpload = false
		session.destinationAddr = "10.0.0.1"
		session.privateKeyPath = "/tmp/key"
		session.instance.InstanceId = strPtr("i-123")

		args := session.buildArgs()

		// Download: remote then local
		assert.Equal(t, "admin@10.0.0.1:/remote/file.txt", args[len(args)-2])
		assert.Equal(t, "/local/", args[len(args)-1])
	})

	t.Run("without login", func(t *testing.T) {
		t.Parallel()

		session := &SCPSession{}
		session.Login = ""
		session.LocalPath = "file.txt"
		session.RemotePath = "/path"
		session.IsUpload = true
		session.destinationAddr = "host"
		session.privateKeyPath = "/tmp/key"
		session.instance.InstanceId = strPtr("i-123")

		args := session.buildArgs()

		// Without login: just host:path
		assert.Equal(t, "host:/path", args[len(args)-1])
	})
}
