package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEICETunnelSession(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args        []string
		wantHost    string
		wantPort    string
		wantEICEID  string
		wantErr     bool
		errIs       error
		errContains string
	}{
		// Valid cases
		"all required flags": {
			args:       []string{"--host", "10.0.0.1", "--port", "22", "--eice-id", "eice-123"},
			wantHost:   "10.0.0.1",
			wantPort:   "22",
			wantEICEID: "eice-123",
		},
		"with region": {
			args:       []string{"--host", "10.0.0.1", "--port", "22", "--eice-id", "eice-123", "--region", "us-west-2"},
			wantHost:   "10.0.0.1",
			wantPort:   "22",
			wantEICEID: "eice-123",
		},
		"with profile": {
			args:       []string{"--host", "10.0.0.1", "--port", "22", "--eice-id", "eice-123", "--profile", "myprofile"},
			wantHost:   "10.0.0.1",
			wantPort:   "22",
			wantEICEID: "eice-123",
		},
		"with debug": {
			args:       []string{"--host", "10.0.0.1", "--port", "22", "--eice-id", "eice-123", "--debug"},
			wantHost:   "10.0.0.1",
			wantPort:   "22",
			wantEICEID: "eice-123",
		},
		"non-standard port": {
			args:       []string{"--host", "10.0.0.1", "--port", "443", "--eice-id", "eice-123"},
			wantHost:   "10.0.0.1",
			wantPort:   "443",
			wantEICEID: "eice-123",
		},

		// Error cases - missing required flags
		"missing host": {
			args:    []string{"--port", "22", "--eice-id", "eice-123"},
			wantErr: true,
			errIs:   ErrMissingHost,
		},
		"missing eice-id": {
			args:    []string{"--host", "10.0.0.1", "--port", "22"},
			wantErr: true,
			errIs:   ErrMissingEICEID,
		},
		"missing port": {
			args:    []string{"--host", "10.0.0.1", "--eice-id", "eice-123"},
			wantErr: true,
			errIs:   ErrMissingPort,
		},
		"all missing": {
			args:    []string{},
			wantErr: true,
			errIs:   ErrMissingHost,
		},

		// Error cases - unexpected arguments
		"unexpected positional": {
			args:        []string{"--host", "10.0.0.1", "--port", "22", "--eice-id", "eice-123", "extra"},
			wantErr:     true,
			errContains: "unexpected argument",
		},
		"unknown flag": {
			args:        []string{"--host", "10.0.0.1", "--port", "22", "--eice-id", "eice-123", "--unknown"},
			wantErr:     true,
			errContains: "unknown option",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			session, err := NewEICETunnelSession(tc.args)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errIs != nil {
					assert.ErrorIs(t, err, tc.errIs)
				}
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, session)

			assert.Equal(t, tc.wantHost, session.Host, "host")
			assert.Equal(t, tc.wantPort, session.Port, "port")
			assert.Equal(t, tc.wantEICEID, session.EICEID, "eiceID")
		})
	}
}

func TestNewSSMTunnelSession(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args           []string
		wantInstanceID string
		wantPort       string
		wantErr        bool
		errIs          error
		errContains    string
	}{
		// Valid cases
		"all required flags": {
			args:           []string{"--instance-id", "i-1234567890abcdef0", "--port", "22"},
			wantInstanceID: "i-1234567890abcdef0",
			wantPort:       "22",
		},
		"with region": {
			args:           []string{"--instance-id", "i-123", "--port", "22", "--region", "us-west-2"},
			wantInstanceID: "i-123",
			wantPort:       "22",
		},
		"with profile": {
			args:           []string{"--instance-id", "i-123", "--port", "22", "--profile", "myprofile"},
			wantInstanceID: "i-123",
			wantPort:       "22",
		},
		"with debug": {
			args:           []string{"--instance-id", "i-123", "--port", "22", "--debug"},
			wantInstanceID: "i-123",
			wantPort:       "22",
		},
		"non-standard port": {
			args:           []string{"--instance-id", "i-123", "--port", "3389"},
			wantInstanceID: "i-123",
			wantPort:       "3389",
		},

		// Error cases - missing required flags
		"missing instance-id": {
			args:    []string{"--port", "22"},
			wantErr: true,
			errIs:   ErrMissingInstanceID,
		},
		"missing port": {
			args:    []string{"--instance-id", "i-123"},
			wantErr: true,
			errIs:   ErrMissingPort,
		},
		"all missing": {
			args:    []string{},
			wantErr: true,
			errIs:   ErrMissingInstanceID,
		},

		// Error cases - unexpected arguments
		"unexpected positional": {
			args:        []string{"--instance-id", "i-123", "--port", "22", "extra"},
			wantErr:     true,
			errContains: "unexpected argument",
		},
		"unknown flag": {
			args:        []string{"--instance-id", "i-123", "--port", "22", "--unknown"},
			wantErr:     true,
			errContains: "unknown option",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			session, err := NewSSMTunnelSession(tc.args)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errIs != nil {
					assert.ErrorIs(t, err, tc.errIs)
				}
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, session)

			assert.Equal(t, tc.wantInstanceID, session.InstanceID, "instanceID")
			assert.Equal(t, tc.wantPort, session.Port, "port")
		})
	}
}
