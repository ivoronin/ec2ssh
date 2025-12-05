package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSSMSession(t *testing.T) {

	tests := []struct {
		name    string
		args    []string
		wantErr string
		check   func(t *testing.T, session *SSMSession)
	}{
		// Basic forms
		{
			name: "basic destination",
			args: []string{"host"},
			check: func(t *testing.T, session *SSMSession) {
				assert.Equal(t, "host", session.Destination)
			},
		},
		{
			name: "user@host destination extracts host",
			args: []string{"user@host"},
			check: func(t *testing.T, session *SSMSession) {
				assert.Equal(t, "host", session.Destination)
			},
		},
		{
			name: "instance ID destination",
			args: []string{"i-0123456789abcdef0"},
			check: func(t *testing.T, session *SSMSession) {
				assert.Equal(t, "i-0123456789abcdef0", session.Destination)
			},
		},

		// AWS options
		{
			name: "--region flag",
			args: []string{"--region", "us-west-2", "host"},
			check: func(t *testing.T, session *SSMSession) {
				assert.Equal(t, "us-west-2", session.Region)
			},
		},
		{
			name: "--profile flag",
			args: []string{"--profile", "prod", "host"},
			check: func(t *testing.T, session *SSMSession) {
				assert.Equal(t, "prod", session.Profile)
			},
		},
		{
			name: "--region and --profile",
			args: []string{"--region", "eu-west-1", "--profile", "myprofile", "host"},
			check: func(t *testing.T, session *SSMSession) {
				assert.Equal(t, "eu-west-1", session.Region)
				assert.Equal(t, "myprofile", session.Profile)
			},
		},

		// Destination types
		{
			name: "destination-type id",
			args: []string{"--destination-type", "id", "i-123"},
		},
		{
			name: "destination-type name_tag",
			args: []string{"--destination-type", "name_tag", "my-server"},
		},
		{
			name:    "destination-type invalid",
			args:    []string{"--destination-type", "invalid", "host"},
			wantErr: "unknown type",
		},

		// Debug
		{
			name: "--debug flag",
			args: []string{"--debug", "host"},
			check: func(t *testing.T, session *SSMSession) {
				assert.True(t, session.Debug)
			},
		},

		// Error cases
		{
			name:    "no destination",
			args:    []string{},
			wantErr: "missing destination",
		},
		{
			name:    "missing value for --region",
			args:    []string{"--region"},
			wantErr: "missing value",
		},
		{
			name:    "missing value for --profile",
			args:    []string{"--profile"},
			wantErr: "missing value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := NewSSMSession(tt.args)

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
