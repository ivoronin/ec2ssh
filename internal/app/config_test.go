package app

import (
	"testing"

	"github.com/ivoronin/ec2ssh/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOptions(t *testing.T) {
	t.Parallel()

	parsedArgs := cli.ParsedArgs{
		Options: map[string]string{
			"-l":         "login",
			"--use-eice": "true",
		},
		Destination:     "host",
		CommandWithArgs: []string{"command"},
		SSHArgs:         []string{"-t"},
	}

	options, err := NewOptions(parsedArgs)

	require.NoError(t, err)
	assert.Equal(t, "host", options.Destination)
	assert.Equal(t, "login", options.Login)
	assert.True(t, options.UseEICE)

	parsedArgs = cli.ParsedArgs{
		Options: map[string]string{
			"--destination-type": "unknown-dst-type",
		},
		Destination: "host",
	}

	_, err = NewOptions(parsedArgs)

	require.Error(t, err)
}
