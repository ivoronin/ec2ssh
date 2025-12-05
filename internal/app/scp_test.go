package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSCPSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr string // empty means no error expected
		check   func(t *testing.T, session *SCPSession)
	}{
		// Download cases
		{
			name: "download basic",
			args: []string{"user@host:/remote/file.txt", "./local/"},
			check: func(t *testing.T, session *SCPSession) {
				assert.Equal(t, "host", session.Destination)
				assert.Equal(t, "user", session.Login)
				assert.Equal(t, "/remote/file.txt", session.RemotePath)
				assert.Equal(t, "./local/", session.LocalPath)
				assert.False(t, session.IsUpload)
			},
		},
		{
			name: "download with instance ID",
			args: []string{"ec2-user@i-0123456789abcdef0:/var/log/app.log", "/tmp/"},
			check: func(t *testing.T, session *SCPSession) {
				assert.Equal(t, "i-0123456789abcdef0", session.Destination)
				assert.Equal(t, "ec2-user", session.Login)
				assert.Equal(t, "/var/log/app.log", session.RemotePath)
				assert.Equal(t, "/tmp/", session.LocalPath)
				assert.False(t, session.IsUpload)
			},
		},

		// Upload cases
		{
			name: "upload basic",
			args: []string{"./local/file.txt", "user@host:/remote/path/"},
			check: func(t *testing.T, session *SCPSession) {
				assert.Equal(t, "host", session.Destination)
				assert.Equal(t, "user", session.Login)
				assert.Equal(t, "/remote/path/", session.RemotePath)
				assert.Equal(t, "./local/file.txt", session.LocalPath)
				assert.True(t, session.IsUpload)
			},
		},
		{
			name: "upload to name tag",
			args: []string{"/etc/config.yaml", "ubuntu@web-prod:/etc/app/"},
			check: func(t *testing.T, session *SCPSession) {
				assert.Equal(t, "web-prod", session.Destination)
				assert.Equal(t, "ubuntu", session.Login)
				assert.Equal(t, "/etc/app/", session.RemotePath)
				assert.Equal(t, "/etc/config.yaml", session.LocalPath)
				assert.True(t, session.IsUpload)
			},
		},

		// Port handling (SCP uses uppercase -P)
		{
			name: "-P flag sets port",
			args: []string{"-P", "2222", "user@host:/path", "./local"},
			check: func(t *testing.T, session *SCPSession) {
				assert.Equal(t, "2222", session.Port)
			},
		},

		// Identity file
		{
			name: "-i flag sets identity file",
			args: []string{"-i", "/path/to/key", "user@host:/path", "./local"},
			check: func(t *testing.T, session *SCPSession) {
				assert.Equal(t, "/path/to/key", session.IdentityFile)
			},
		},

		// EICE options
		{
			name: "--use-eice",
			args: []string{"--use-eice", "user@host:/path", "./local"},
			check: func(t *testing.T, session *SCPSession) {
				assert.True(t, session.UseEICE)
			},
		},
		{
			name: "--eice-id implies --use-eice",
			args: []string{"--eice-id", "eice-12345678", "user@host:/path", "./local"},
			check: func(t *testing.T, session *SCPSession) {
				assert.Equal(t, "eice-12345678", session.EICEID)
				assert.True(t, session.UseEICE)
			},
		},

		// Other flags
		{
			name: "--no-send-keys",
			args: []string{"--no-send-keys", "user@host:/path", "./local"},
			check: func(t *testing.T, session *SCPSession) {
				assert.True(t, session.NoSendKeys)
			},
		},
		{
			name: "--debug",
			args: []string{"--debug", "user@host:/path", "./local"},
			check: func(t *testing.T, session *SCPSession) {
				assert.True(t, session.Debug)
			},
		},
		{
			name: "--use-ssm",
			args: []string{"--use-ssm", "user@host:/path", "./local"},
			check: func(t *testing.T, session *SCPSession) {
				assert.True(t, session.UseSSM)
			},
		},
		{
			name: "--region and --profile",
			args: []string{"--region", "us-west-2", "--profile", "myprofile", "user@host:/path", "./local"},
			check: func(t *testing.T, session *SCPSession) {
				assert.Equal(t, "us-west-2", session.Region)
				assert.Equal(t, "myprofile", session.Profile)
			},
		},

		// Passthrough options
		{
			name: "scp passthrough -o",
			args: []string{"-o", "StrictHostKeyChecking=no", "user@host:/path", "./local"},
			check: func(t *testing.T, session *SCPSession) {
				assert.Equal(t, []string{"-o", "StrictHostKeyChecking=no"}, session.PassArgs)
			},
		},
		{
			name: "scp passthrough -r (recursive)",
			args: []string{"-r", "user@host:/path", "./local"},
			check: func(t *testing.T, session *SCPSession) {
				assert.Contains(t, session.PassArgs, "-r")
			},
		},

		// Type validations
		{
			name: "address-type private",
			args: []string{"--address-type", "private", "user@host:/path", "./local"},
		},
		{
			name:    "address-type invalid",
			args:    []string{"--address-type", "invalid", "user@host:/path", "./local"},
			wantErr: "unknown type",
		},
		{
			name: "destination-type id",
			args: []string{"--destination-type", "id", "user@host:/path", "./local"},
		},
		{
			name:    "destination-type invalid",
			args:    []string{"--destination-type", "invalid", "user@host:/path", "./local"},
			wantErr: "unknown type",
		},

		// Error cases - SCP operand parsing
		{
			name:    "no operands",
			args:    []string{},
			wantErr: "requires exactly 2 operands",
		},
		{
			name:    "one operand",
			args:    []string{"user@host:/path"},
			wantErr: "requires exactly 2 operands",
		},
		{
			name:    "both local",
			args:    []string{"/local/path1", "/local/path2"},
			wantErr: "no remote operand",
		},
		{
			name:    "both remote",
			args:    []string{"host1:/path1", "host2:/path2"},
			wantErr: "multiple remote operands",
		},
		{
			name:    "empty remote path",
			args:    []string{"host:", "./local"},
			wantErr: "remote path cannot be empty",
		},
		{
			name:    "missing value for --region",
			args:    []string{"--region"},
			wantErr: "missing value",
		},

		// Edge cases
		{
			name: "local file with colon in name (dot prefix)",
			args: []string{"./file:with:colons", "host:/remote"},
			check: func(t *testing.T, session *SCPSession) {
				assert.Equal(t, "./file:with:colons", session.LocalPath)
				assert.Equal(t, "host", session.Destination)
				assert.Equal(t, "/remote", session.RemotePath)
				assert.True(t, session.IsUpload)
			},
		},
		{
			name: "remote without user defaults to current user",
			args: []string{"host:/path", "./local"},
			check: func(t *testing.T, session *SCPSession) {
				assert.Equal(t, "host", session.Destination)
				assert.NotEmpty(t, session.Login) // defaults to current user
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session, err := NewSCPSession(tt.args)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, session)
			}
		})
	}
}
