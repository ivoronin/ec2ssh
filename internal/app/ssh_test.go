package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSSHSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr string // empty means no error expected
		check   func(t *testing.T, session *SSHSession)
	}{
		{
			name: "basic with login and use-eice",
			args: []string{"-l", "login", "--use-eice", "-t", "host", "command"},
			check: func(t *testing.T, session *SSHSession) {
				assert.Equal(t, "host", session.Destination)
				assert.Equal(t, "login", session.Login)
				assert.True(t, session.UseEICE)
				assert.Equal(t, []string{"command"}, session.CommandWithArgs)
				assert.Equal(t, []string{"-t"}, session.PassArgs)
			},
		},
		{
			name: "ssh passthrough -L",
			args: []string{"-L", "8080:localhost:80", "host"},
			check: func(t *testing.T, session *SSHSession) {
				assert.Equal(t, "host", session.Destination)
				assert.Equal(t, []string{"-L", "8080:localhost:80"}, session.PassArgs)
			},
		},
		{
			name: "destination with user@host",
			args: []string{"user@host"},
			check: func(t *testing.T, session *SSHSession) {
				assert.Equal(t, "host", session.Destination)
				assert.Equal(t, "user", session.Login)
			},
		},
		{
			name: "destination ssh:// URL with port",
			args: []string{"ssh://user@host:2222"},
			check: func(t *testing.T, session *SSHSession) {
				assert.Equal(t, "host", session.Destination)
				assert.Equal(t, "user", session.Login)
				assert.Equal(t, "2222", session.Port)
			},
		},
		{
			name: "-l flag overrides user@ in destination",
			args: []string{"-l", "flaguser", "destuser@host"},
			check: func(t *testing.T, session *SSHSession) {
				assert.Equal(t, "host", session.Destination)
				assert.Equal(t, "flaguser", session.Login)
			},
		},
		{
			name: "-p flag overrides port in ssh:// URL",
			args: []string{"-p", "3333", "ssh://user@host:2222"},
			check: func(t *testing.T, session *SSHSession) {
				assert.Equal(t, "host", session.Destination)
				assert.Equal(t, "3333", session.Port)
			},
		},
		{
			name: "--eice-id implies --use-eice",
			args: []string{"--eice-id", "eice-12345678", "host"},
			check: func(t *testing.T, session *SSHSession) {
				assert.Equal(t, "eice-12345678", session.EICEID)
				assert.True(t, session.UseEICE)
			},
		},
		{
			name: "--no-send-keys",
			args: []string{"--no-send-keys", "host"},
			check: func(t *testing.T, session *SSHSession) {
				assert.True(t, session.NoSendKeys)
			},
		},
		{
			name: "--debug",
			args: []string{"--debug", "host"},
			check: func(t *testing.T, session *SSHSession) {
				assert.True(t, session.Debug)
			},
		},
		{
			name: "--region and --profile",
			args: []string{"--region", "us-west-2", "--profile", "myprofile", "host"},
			check: func(t *testing.T, session *SSHSession) {
				assert.Equal(t, "us-west-2", session.Region)
				assert.Equal(t, "myprofile", session.Profile)
			},
		},
		{
			name:    "no destination",
			args:    []string{"--use-eice"},
			wantErr: "missing destination",
		},
		// Address types
		{
			name: "address-type private",
			args: []string{"--address-type", "private", "host"},
		},
		{
			name: "address-type public",
			args: []string{"--address-type", "public", "host"},
		},
		{
			name: "address-type ipv6",
			args: []string{"--address-type", "ipv6", "host"},
		},
		{
			name:    "address-type invalid",
			args:    []string{"--address-type", "invalid", "host"},
			wantErr: "unknown type",
		},
		// Destination types
		{
			name: "destination-type id",
			args: []string{"--destination-type", "id", "host"},
		},
		{
			name: "destination-type private_ip",
			args: []string{"--destination-type", "private_ip", "host"},
		},
		{
			name: "destination-type public_ip",
			args: []string{"--destination-type", "public_ip", "host"},
		},
		{
			name: "destination-type ipv6",
			args: []string{"--destination-type", "ipv6", "host"},
		},
		{
			name: "destination-type private_dns",
			args: []string{"--destination-type", "private_dns", "host"},
		},
		{
			name: "destination-type name_tag",
			args: []string{"--destination-type", "name_tag", "host"},
		},
		{
			name:    "destination-type invalid",
			args:    []string{"--destination-type", "invalid", "host"},
			wantErr: "unknown type",
		},
		{
			name:    "missing value for --region",
			args:    []string{"--region"},
			wantErr: "missing value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session, err := NewSSHSession(tt.args)

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
