package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOptions(t *testing.T) {
	t.Parallel()

	args := []string{"-l", "login", "--use-eice", "-t", "host", "command"}
	options, err := NewOptions(args)

	require.NoError(t, err)
	assert.Equal(t, "host", options.Destination)
	assert.Equal(t, "login", options.Login)
	assert.True(t, options.UseEICE)
	assert.Equal(t, []string{"command"}, options.CommandWithArgs)
	assert.Equal(t, []string{"-t"}, options.SSHArgs)
}

func TestNewOptions_UnknownDstType(t *testing.T) {
	t.Parallel()

	args := []string{"--destination-type", "unknown-dst-type", "host"}
	_, err := NewOptions(args)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown type")
}

func TestNewOptions_ListMode(t *testing.T) {
	t.Parallel()

	args := []string{"--list", "--region", "us-west-2"}
	options, err := NewOptions(args)

	require.NoError(t, err)
	assert.True(t, options.DoList)
	assert.Equal(t, "us-west-2", options.Region)
}

func TestNewOptions_ListModeWithDisallowedOption(t *testing.T) {
	t.Parallel()

	args := []string{"--list", "--use-eice"}
	_, err := NewOptions(args)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--list")
}

func TestNewOptions_ListColumnsWithoutList(t *testing.T) {
	t.Parallel()

	args := []string{"--list-columns", "id,name", "host"}
	_, err := NewOptions(args)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--list-columns")
}

func TestNewOptions_SSHPassthrough(t *testing.T) {
	t.Parallel()

	// -L takes a value, should be passed through correctly
	args := []string{"-L", "8080:localhost:80", "host"}
	options, err := NewOptions(args)

	require.NoError(t, err)
	assert.Equal(t, "host", options.Destination)
	assert.Equal(t, []string{"-L", "8080:localhost:80"}, options.SSHArgs)
}

func TestNewOptions_DestinationWithUser(t *testing.T) {
	t.Parallel()

	args := []string{"user@host"}
	options, err := NewOptions(args)

	require.NoError(t, err)
	assert.Equal(t, "host", options.Destination)
	assert.Equal(t, "user", options.Login)
}

func TestNewOptions_DestinationSSHURL(t *testing.T) {
	t.Parallel()

	// ssh:// URL format supports port parsing
	args := []string{"ssh://user@host:2222"}
	options, err := NewOptions(args)

	require.NoError(t, err)
	assert.Equal(t, "host", options.Destination)
	assert.Equal(t, "user", options.Login)
	assert.Equal(t, "2222", options.Port)
}

func TestNewOptions_FlagOverridesDestination(t *testing.T) {
	t.Parallel()

	// -l flag should take precedence over user@ in destination
	args := []string{"-l", "flaguser", "destuser@host"}
	options, err := NewOptions(args)

	require.NoError(t, err)
	assert.Equal(t, "host", options.Destination)
	assert.Equal(t, "flaguser", options.Login)
}

func TestNewOptions_PortFlagOverridesDestination(t *testing.T) {
	t.Parallel()

	// -p flag should take precedence over port in ssh:// URL
	args := []string{"-p", "3333", "ssh://user@host:2222"}
	options, err := NewOptions(args)

	require.NoError(t, err)
	assert.Equal(t, "host", options.Destination)
	assert.Equal(t, "3333", options.Port)
}

func TestNewOptions_AddressType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		addrType string
		wantErr  bool
	}{
		{"private", "private", false},
		{"public", "public", false},
		{"ipv6", "ipv6", false},
		{"empty (auto)", "", false},
		{"unknown", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			args := []string{"host"}
			if tt.addrType != "" {
				args = []string{"--address-type", tt.addrType, "host"}
			}

			_, err := NewOptions(args)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unknown type")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewOptions_EICEID(t *testing.T) {
	t.Parallel()

	args := []string{"--eice-id", "eice-12345678", "host"}
	options, err := NewOptions(args)

	require.NoError(t, err)
	assert.Equal(t, "eice-12345678", options.EICEID)
	assert.True(t, options.UseEICE) // UseEICE should be auto-enabled when EICEID is set
}

func TestNewOptions_Help(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{"short flag", []string{"-h"}},
		{"long flag", []string{"--help"}},
		{"with other args", []string{"--region", "us-west-2", "--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewOptions(tt.args)

			require.ErrorIs(t, err, ErrHelp)
		})
	}
}
