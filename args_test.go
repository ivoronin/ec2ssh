package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseArgs(t *testing.T) {
	t.Parallel()

	parsedArgs, err := ParseArgs([]string{"host"})

	require.NoError(t, err)
	assert.Equal(t, "host", parsedArgs.Destination)
	assert.Equal(t, map[string]string{}, parsedArgs.Options)
	assert.Equal(t, []string{}, parsedArgs.CommandWithArgs)
	assert.Equal(t, []string{}, parsedArgs.SSHArgs)

	parsedArgs, err = ParseArgs([]string{"host", "command"})

	require.NoError(t, err)
	assert.Equal(t, "host", parsedArgs.Destination)
	assert.Equal(t, map[string]string{}, parsedArgs.Options)
	assert.Equal(t, []string{"command"}, parsedArgs.CommandWithArgs)
	assert.Equal(t, []string{}, parsedArgs.SSHArgs)

	/* consumed short options */
	parsedArgs, err = ParseArgs([]string{"-l", "login", "host", "command"})

	require.NoError(t, err)
	assert.Equal(t, "host", parsedArgs.Destination)
	assert.Equal(t, map[string]string{"-l": "login"}, parsedArgs.Options)
	assert.Equal(t, []string{"command"}, parsedArgs.CommandWithArgs)
	assert.Equal(t, []string{}, parsedArgs.SSHArgs)

	/* host before options */
	parsedArgs, err = ParseArgs([]string{"host", "-l", "login", "command"})

	require.NoError(t, err)
	assert.Equal(t, "host", parsedArgs.Destination)
	assert.Equal(t, map[string]string{"-l": "login"}, parsedArgs.Options)
	assert.Equal(t, []string{"command"}, parsedArgs.CommandWithArgs)
	assert.Equal(t, []string{}, parsedArgs.SSHArgs)

	/* more consumed short options */
	parsedArgs, err = ParseArgs([]string{"host", "-l", "login", "-p", "port", "command"})

	require.NoError(t, err)
	assert.Equal(t, "host", parsedArgs.Destination)
	assert.Equal(t, map[string]string{"-l": "login", "-p": "port"}, parsedArgs.Options)
	assert.Equal(t, []string{"command"}, parsedArgs.CommandWithArgs)
	assert.Equal(t, []string{}, parsedArgs.SSHArgs)

	/* command with short options */
	parsedArgs, err = ParseArgs([]string{"host", "-l", "login", "command", "-p", "port"})

	require.NoError(t, err)
	assert.Equal(t, "host", parsedArgs.Destination)
	assert.Equal(t, map[string]string{"-l": "login"}, parsedArgs.Options)
	assert.Equal(t, []string{"command", "-p", "port"}, parsedArgs.CommandWithArgs)
	assert.Equal(t, []string{}, parsedArgs.SSHArgs)

	/* command starts with a double dash */
	_, err = ParseArgs([]string{"host", "-l", "login", "--command", "-p", "port"})

	require.Error(t, err)

	/* command starts with a double dash after a double dash */
	parsedArgs, err = ParseArgs([]string{"host", "-l", "login", "--", "--command", "-p", "port"})
	require.NoError(t, err)
	assert.Equal(t, "host", parsedArgs.Destination)
	assert.Equal(t, map[string]string{"-l": "login"}, parsedArgs.Options)
	assert.Equal(t, []string{"--command", "-p", "port"}, parsedArgs.CommandWithArgs)
	assert.Equal(t, []string{}, parsedArgs.SSHArgs)

	/* unconsumed short options */
	parsedArgs, err = ParseArgs([]string{"host", "-l", "login", "-t", "command", "-p", "port"})
	require.NoError(t, err)
	assert.Equal(t, "host", parsedArgs.Destination)
	assert.Equal(t, map[string]string{"-l": "login"}, parsedArgs.Options)
	assert.Equal(t, []string{"command", "-p", "port"}, parsedArgs.CommandWithArgs)
	assert.Equal(t, []string{"-t"}, parsedArgs.SSHArgs)

	/* illegal SSH option -H */
	parsedArgs, err = ParseArgs([]string{"host", "-l", "login", "-H", "-t", "command", "-p", "port"})
	require.NoError(t, err)
	assert.Equal(t, "host", parsedArgs.Destination)
	assert.Equal(t, map[string]string{"-l": "login"}, parsedArgs.Options)
	assert.Equal(t, []string{"command", "-p", "port"}, parsedArgs.CommandWithArgs)
	assert.Equal(t, []string{"-H", "-t"}, parsedArgs.SSHArgs)

	/* illegal SSH option -H with an argument should break parsing */
	parsedArgs, err = ParseArgs([]string{"host", "-l", "login", "-H", "needle", "-t", "command", "-p", "port"})
	require.NoError(t, err)
	assert.Equal(t, "host", parsedArgs.Destination)
	assert.Equal(t, map[string]string{"-l": "login"}, parsedArgs.Options)
	assert.Equal(t, []string{"needle", "-t", "command", "-p", "port"}, parsedArgs.CommandWithArgs)
	assert.Equal(t, []string{"-H"}, parsedArgs.SSHArgs)

	/* passing unconsumed SSH options with arguments */
	parsedArgs, err = ParseArgs([]string{"host", "-J", "jump", "-l", "login", "-t", "command", "-p", "port"})
	require.NoError(t, err)
	assert.Equal(t, "host", parsedArgs.Destination)
	assert.Equal(t, map[string]string{"-l": "login"}, parsedArgs.Options)
	assert.Equal(t, []string{"command", "-p", "port"}, parsedArgs.CommandWithArgs)
	assert.Equal(t, []string{"-Jjump", "-t"}, parsedArgs.SSHArgs)

	/* multiple -l options */
	parsedArgs, err = ParseArgs([]string{"host", "-l", "login", "-l", "login2", "-t", "command", "-p", "port"})
	require.NoError(t, err)
	assert.Equal(t, "host", parsedArgs.Destination)
	assert.Equal(t, map[string]string{"-l": "login"}, parsedArgs.Options)
	assert.Equal(t, []string{"command", "-p", "port"}, parsedArgs.CommandWithArgs)
	assert.Equal(t, []string{"-t"}, parsedArgs.SSHArgs)

	/* boolean long options */
	parsedArgs, err = ParseArgs([]string{"host", "-l", "login", "--use-eice", "-t", "command", "-p", "port"})
	require.NoError(t, err)
	assert.Equal(t, "host", parsedArgs.Destination)
	assert.Equal(t, map[string]string{"-l": "login", "--use-eice": "true"}, parsedArgs.Options)
	assert.Equal(t, []string{"command", "-p", "port"}, parsedArgs.CommandWithArgs)
	assert.Equal(t, []string{"-t"}, parsedArgs.SSHArgs)

	/* boolean and string long options */
	parsedArgs, err = ParseArgs([]string{"host", "-l", "login", "--use-eice", "--profile", "profile"})
	require.NoError(t, err)
	assert.Equal(t, "host", parsedArgs.Destination)
	assert.Equal(t, map[string]string{"-l": "login", "--use-eice": "true", "--profile": "profile"}, parsedArgs.Options)
	assert.Equal(t, []string{}, parsedArgs.CommandWithArgs)
	assert.Equal(t, []string{}, parsedArgs.SSHArgs)
}
