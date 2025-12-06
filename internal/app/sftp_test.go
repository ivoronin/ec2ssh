package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSFTPSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr string // empty means no error expected
		check   func(t *testing.T, session *SFTPSession)
	}{
		// Basic forms
		{
			name: "basic user@host:path",
			args: []string{"user@host:/remote/path"},
			check: func(t *testing.T, session *SFTPSession) {
				assert.Equal(t, "host", session.Destination)
				assert.Equal(t, "user", session.Login)
				assert.Equal(t, "/remote/path", session.RemotePath)
			},
		},
		{
			name: "host only defaults to current user",
			args: []string{"host"},
			check: func(t *testing.T, session *SFTPSession) {
				assert.Equal(t, "host", session.Destination)
				assert.NotEmpty(t, session.Login) // defaults to current user
			},
		},
		{
			name: "sftp:// URL with port and path",
			args: []string{"sftp://user@host:2222/remote/path"},
			check: func(t *testing.T, session *SFTPSession) {
				assert.Equal(t, "host", session.Destination)
				assert.Equal(t, "user", session.Login)
				assert.Equal(t, "2222", session.Port)
				assert.Equal(t, "remote/path", session.RemotePath)
			},
		},

		// Port handling (SFTP uses uppercase -P)
		{
			name: "-P flag sets port",
			args: []string{"-P", "3333", "user@host"},
			check: func(t *testing.T, session *SFTPSession) {
				assert.Equal(t, "3333", session.Port)
			},
		},
		{
			name: "-P flag overrides port in sftp:// URL",
			args: []string{"-P", "3333", "sftp://user@host:2222"},
			check: func(t *testing.T, session *SFTPSession) {
				assert.Equal(t, "3333", session.Port)
			},
		},

		// Identity file
		{
			name: "-i flag sets identity file",
			args: []string{"-i", "/path/to/key", "user@host"},
			check: func(t *testing.T, session *SFTPSession) {
				assert.Equal(t, "/path/to/key", session.IdentityFile)
			},
		},

		// EICE options
		{
			name: "--use-eice",
			args: []string{"--use-eice", "user@host"},
			check: func(t *testing.T, session *SFTPSession) {
				assert.True(t, session.UseEICE)
			},
		},
		{
			name: "--eice-id implies --use-eice",
			args: []string{"--eice-id", "eice-12345678", "user@host"},
			check: func(t *testing.T, session *SFTPSession) {
				assert.Equal(t, "eice-12345678", session.EICEID)
				assert.True(t, session.UseEICE)
			},
		},

		// Other flags
		{
			name: "--no-send-keys",
			args: []string{"--no-send-keys", "user@host"},
			check: func(t *testing.T, session *SFTPSession) {
				assert.True(t, session.NoSendKeys)
			},
		},
		{
			name: "--debug",
			args: []string{"--debug", "user@host"},
			check: func(t *testing.T, session *SFTPSession) {
				assert.True(t, session.Debug)
			},
		},
		{
			name: "--use-ssm",
			args: []string{"--use-ssm", "user@host"},
			check: func(t *testing.T, session *SFTPSession) {
				assert.True(t, session.UseSSM)
			},
		},
		{
			name: "--region and --profile",
			args: []string{"--region", "us-west-2", "--profile", "myprofile", "user@host"},
			check: func(t *testing.T, session *SFTPSession) {
				assert.Equal(t, "us-west-2", session.Region)
				assert.Equal(t, "myprofile", session.Profile)
			},
		},

		// Passthrough options
		{
			name: "sftp passthrough -B",
			args: []string{"-B", "32768", "user@host"},
			check: func(t *testing.T, session *SFTPSession) {
				assert.Equal(t, []string{"-B", "32768"}, session.PassArgs)
			},
		},
		{
			name: "sftp passthrough -o",
			args: []string{"-o", "StrictHostKeyChecking=no", "user@host"},
			check: func(t *testing.T, session *SFTPSession) {
				assert.Equal(t, []string{"-o", "StrictHostKeyChecking=no"}, session.PassArgs)
			},
		},

		// Type validations
		{
			name: "address-type private",
			args: []string{"--address-type", "private", "user@host"},
		},
		{
			name: "address-type public",
			args: []string{"--address-type", "public", "user@host"},
		},
		{
			name: "address-type ipv6",
			args: []string{"--address-type", "ipv6", "user@host"},
		},
		{
			name:    "address-type invalid",
			args:    []string{"--address-type", "invalid", "user@host"},
			wantErr: "unknown",
		},
		{
			name: "destination-type id",
			args: []string{"--destination-type", "id", "user@host"},
		},
		{
			name: "destination-type name_tag",
			args: []string{"--destination-type", "name_tag", "user@host"},
		},
		{
			name:    "destination-type invalid",
			args:    []string{"--destination-type", "invalid", "user@host"},
			wantErr: "unknown",
		},

		// Error cases
		{
			name:    "missing value for --region",
			args:    []string{"--region"},
			wantErr: "missing value",
		},
		{
			name:    "no destination",
			args:    []string{"--use-eice"},
			wantErr: "missing destination",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session, err := NewSFTPSession(tt.args)

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
