package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSCPOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr string // empty means no error expected
		check   func(t *testing.T, opts *SCPOptions)
	}{
		// Download cases
		{
			name: "download basic",
			args: []string{"user@host:/remote/file.txt", "./local/"},
			check: func(t *testing.T, opts *SCPOptions) {
				assert.Equal(t, "host", opts.Destination)
				assert.Equal(t, "user", opts.Login)
				assert.Equal(t, "/remote/file.txt", opts.RemotePath)
				assert.Equal(t, "./local/", opts.LocalPath)
				assert.False(t, opts.IsUpload)
			},
		},
		{
			name: "download with instance ID",
			args: []string{"ec2-user@i-0123456789abcdef0:/var/log/app.log", "/tmp/"},
			check: func(t *testing.T, opts *SCPOptions) {
				assert.Equal(t, "i-0123456789abcdef0", opts.Destination)
				assert.Equal(t, "ec2-user", opts.Login)
				assert.Equal(t, "/var/log/app.log", opts.RemotePath)
				assert.Equal(t, "/tmp/", opts.LocalPath)
				assert.False(t, opts.IsUpload)
			},
		},

		// Upload cases
		{
			name: "upload basic",
			args: []string{"./local/file.txt", "user@host:/remote/path/"},
			check: func(t *testing.T, opts *SCPOptions) {
				assert.Equal(t, "host", opts.Destination)
				assert.Equal(t, "user", opts.Login)
				assert.Equal(t, "/remote/path/", opts.RemotePath)
				assert.Equal(t, "./local/file.txt", opts.LocalPath)
				assert.True(t, opts.IsUpload)
			},
		},
		{
			name: "upload to name tag",
			args: []string{"/etc/config.yaml", "ubuntu@web-prod:/etc/app/"},
			check: func(t *testing.T, opts *SCPOptions) {
				assert.Equal(t, "web-prod", opts.Destination)
				assert.Equal(t, "ubuntu", opts.Login)
				assert.Equal(t, "/etc/app/", opts.RemotePath)
				assert.Equal(t, "/etc/config.yaml", opts.LocalPath)
				assert.True(t, opts.IsUpload)
			},
		},

		// Port handling (SCP uses uppercase -P)
		{
			name: "-P flag sets port",
			args: []string{"-P", "2222", "user@host:/path", "./local"},
			check: func(t *testing.T, opts *SCPOptions) {
				assert.Equal(t, "2222", opts.Port)
			},
		},

		// Identity file
		{
			name: "-i flag sets identity file",
			args: []string{"-i", "/path/to/key", "user@host:/path", "./local"},
			check: func(t *testing.T, opts *SCPOptions) {
				assert.Equal(t, "/path/to/key", opts.IdentityFile)
			},
		},

		// EICE options
		{
			name: "--use-eice",
			args: []string{"--use-eice", "user@host:/path", "./local"},
			check: func(t *testing.T, opts *SCPOptions) {
				assert.True(t, opts.UseEICE)
			},
		},
		{
			name: "--eice-id implies --use-eice",
			args: []string{"--eice-id", "eice-12345678", "user@host:/path", "./local"},
			check: func(t *testing.T, opts *SCPOptions) {
				assert.Equal(t, "eice-12345678", opts.EICEID)
				assert.True(t, opts.UseEICE)
			},
		},

		// Other flags
		{
			name: "--no-send-keys",
			args: []string{"--no-send-keys", "user@host:/path", "./local"},
			check: func(t *testing.T, opts *SCPOptions) {
				assert.True(t, opts.NoSendKeys)
			},
		},
		{
			name: "--debug",
			args: []string{"--debug", "user@host:/path", "./local"},
			check: func(t *testing.T, opts *SCPOptions) {
				assert.True(t, opts.Debug)
			},
		},
		{
			name: "--region and --profile",
			args: []string{"--region", "us-west-2", "--profile", "myprofile", "user@host:/path", "./local"},
			check: func(t *testing.T, opts *SCPOptions) {
				assert.Equal(t, "us-west-2", opts.Region)
				assert.Equal(t, "myprofile", opts.Profile)
			},
		},

		// Passthrough options
		{
			name: "scp passthrough -o",
			args: []string{"-o", "StrictHostKeyChecking=no", "user@host:/path", "./local"},
			check: func(t *testing.T, opts *SCPOptions) {
				assert.Equal(t, []string{"-o", "StrictHostKeyChecking=no"}, opts.SCPArgs)
			},
		},
		{
			name: "scp passthrough -r (recursive)",
			args: []string{"-r", "user@host:/path", "./local"},
			check: func(t *testing.T, opts *SCPOptions) {
				assert.Contains(t, opts.SCPArgs, "-r")
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
			check: func(t *testing.T, opts *SCPOptions) {
				assert.Equal(t, "./file:with:colons", opts.LocalPath)
				assert.Equal(t, "host", opts.Destination)
				assert.Equal(t, "/remote", opts.RemotePath)
				assert.True(t, opts.IsUpload)
			},
		},
		{
			name: "remote without user defaults to current user",
			args: []string{"host:/path", "./local"},
			check: func(t *testing.T, opts *SCPOptions) {
				assert.Equal(t, "host", opts.Destination)
				assert.NotEmpty(t, opts.Login) // defaults to current user
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts, err := NewSCPOptions(tt.args)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, opts)
			}
		})
	}
}
