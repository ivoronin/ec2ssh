package app

import (
	"testing"

	"github.com/ivoronin/ec2ssh/internal/ec2client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSSMSession(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args        []string
		wantHost    string
		wantDstType *ec2client.DstType // nil = auto-detect (default)
		wantErr     bool
		errContains string
	}{
		// Basic formats
		"instance id": {
			args:     []string{"i-1234567890abcdef0"},
			wantHost: "i-1234567890abcdef0",
		},
		"user@instance id - user ignored": {
			args:     []string{"ec2-user@i-1234567890abcdef0"},
			wantHost: "i-1234567890abcdef0",
		},
		"private ip": {
			args:     []string{"10.0.0.1"},
			wantHost: "10.0.0.1",
		},
		"name tag": {
			args:     []string{"my-server"},
			wantHost: "my-server",
		},

		// With flags
		"with region": {
			args:     []string{"--region", "us-west-2", "i-123"},
			wantHost: "i-123",
		},
		"with profile": {
			args:     []string{"--profile", "myprofile", "i-123"},
			wantHost: "i-123",
		},
		"with destination type id": {
			args:        []string{"--destination-type", "id", "i-123"},
			wantHost:    "i-123",
			wantDstType: dstTypePtr(ec2client.DstTypeID),
		},
		"with destination type name_tag": {
			args:        []string{"--destination-type", "name_tag", "my-server"},
			wantHost:    "my-server",
			wantDstType: dstTypePtr(ec2client.DstTypeNameTag),
		},
		"with debug": {
			args:     []string{"--debug", "i-123"},
			wantHost: "i-123",
		},

		// Error cases
		"missing destination": {
			args:        []string{},
			wantErr:     true,
			errContains: "missing destination",
		},
		"extra positional argument": {
			args:        []string{"i-123", "extra"},
			wantErr:     true,
			errContains: "unexpected argument",
		},
		"invalid destination type": {
			args:        []string{"--destination-type", "invalid", "i-123"},
			wantErr:     true,
			errContains: "unknown destination type",
		},
		"unknown flag": {
			args:        []string{"--unknown-flag", "i-123"},
			wantErr:     true,
			errContains: "unknown option",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			session, err := NewSSMSession(tc.args)

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
			assert.Equal(t, tc.wantDstType, session.DstType, "dstType")
		})
	}
}

func TestNewSSMSession_NoPassthroughArgs(t *testing.T) {
	t.Parallel()

	// SSM sessions use strict parsing - unknown flags should be rejected
	_, err := NewSSMSession([]string{"-v", "i-123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown option")
}
