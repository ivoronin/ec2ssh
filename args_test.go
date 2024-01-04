package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSSHDestionation(t *testing.T) {
	t.Parallel()

	/* URL like destination */
	login, host, port := parseSSHDestination("ssh://login@host:port")

	assert.Equal(t, "login", login)
	assert.Equal(t, "host", host)
	assert.Equal(t, "port", port)

	login, host, port = parseSSHDestination("ssh://host:port")

	assert.Equal(t, "", login)
	assert.Equal(t, "host", host)
	assert.Equal(t, "port", port)

	login, host, port = parseSSHDestination("ssh://host")

	assert.Equal(t, "", login)
	assert.Equal(t, "host", host)
	assert.Equal(t, "", port)

	login, host, port = parseSSHDestination("ssh://login@domain@host:port")

	assert.Equal(t, "login@domain", login)
	assert.Equal(t, "host", host)
	assert.Equal(t, "port", port)

	login, host, port = parseSSHDestination("ssh://login@domain@[fec1::1]:port")

	assert.Equal(t, "login@domain", login)
	assert.Equal(t, "fec1::1", host)
	assert.Equal(t, "port", port)

	login, host, port = parseSSHDestination("ssh://login@domain@[fec1::1]")

	assert.Equal(t, "login@domain", login)
	assert.Equal(t, "fec1::1", host)
	assert.Equal(t, "", port)

	login, host, port = parseSSHDestination("ssh://@[fec1::1]")

	assert.Equal(t, "", login)
	assert.Equal(t, "fec1::1", host)
	assert.Equal(t, "", port)

	login, host, port = parseSSHDestination("[fec1::1]")

	assert.Equal(t, "", login)
	assert.Equal(t, "[fec1::1]", host)
	assert.Equal(t, "", port)

	/* Non-URL like destination */
	login, host, port = parseSSHDestination("login@host")
	assert.Equal(t, "login", login)
	assert.Equal(t, "host", host)
	assert.Equal(t, "", port)

	login, host, port = parseSSHDestination("host")
	assert.Equal(t, "", login)
	assert.Equal(t, "host", host)
	assert.Equal(t, "", port)

	login, host, port = parseSSHDestination("host:port")
	assert.Equal(t, "", login)
	assert.Equal(t, "host:port", host)
	assert.Equal(t, "", port)

	login, host, port = parseSSHDestination("login@host:port")
	assert.Equal(t, "login", login)
	assert.Equal(t, "host:port", host)
	assert.Equal(t, "", port)
}

func TestParseSSHArgs(t *testing.T) {
	t.Parallel()

	sshArgs, err := ParseSSHArgs([]string{"login@host"})

	require.NoError(t, err)
	assert.Equal(t, "login", sshArgs.Login())
	assert.Equal(t, "host", sshArgs.Destination())

	sshArgs, err = ParseSSHArgs([]string{"-l", "login", "host"})

	require.NoError(t, err)
	assert.Equal(t, "login", sshArgs.Login())
	assert.Equal(t, "host", sshArgs.Destination())

	sshArgs, err = ParseSSHArgs([]string{"-llogin", "host"})

	require.NoError(t, err)
	assert.Equal(t, "login", sshArgs.Login())
	assert.Equal(t, "host", sshArgs.Destination())

	sshArgs, err = ParseSSHArgs([]string{"-llogin", "-p", "port", "host", "command", "arg1", "arg2"})

	require.NoError(t, err)
	assert.Equal(t, "login", sshArgs.Login())
	assert.Equal(t, "host", sshArgs.Destination())
	assert.Equal(t, "port", sshArgs.Port())
	assert.Equal(t, []string{}, sshArgs.otherFlags)
	assert.Equal(t, []string{"command", "arg1", "arg2"}, sshArgs.commandAndArgs)
	assert.Equal(t, []string{"-llogin", "-pport", "host", "command", "arg1", "arg2"}, sshArgs.Args())

	sshArgs, err = ParseSSHArgs([]string{"-llogin", "host", "-p", "port", "command", "arg1", "arg2"})

	require.NoError(t, err)
	assert.Equal(t, "login", sshArgs.Login())
	assert.Equal(t, "host", sshArgs.Destination())
	assert.Equal(t, "port", sshArgs.Port())
	assert.Equal(t, []string{}, sshArgs.otherFlags)
	assert.Equal(t, []string{"command", "arg1", "arg2"}, sshArgs.commandAndArgs)
	assert.Equal(t, []string{"-llogin", "-pport", "host", "command", "arg1", "arg2"}, sshArgs.Args())

	sshArgs, err = ParseSSHArgs([]string{"-llogin", "-X", "-o", "Option=2", "host", "-p", "port", "command", "-l", "arg2"})

	require.NoError(t, err)
	assert.Equal(t, "login", sshArgs.Login())
	assert.Equal(t, "host", sshArgs.Destination())
	assert.Equal(t, "port", sshArgs.Port())
	assert.Equal(t, []string{"-X", "-o", "Option=2"}, sshArgs.otherFlags)
	assert.Equal(t, []string{"command", "-l", "arg2"}, sshArgs.commandAndArgs)
	assert.Equal(t, []string{"-llogin", "-pport", "-X", "-o", "Option=2", "host", "command", "-l", "arg2"}, sshArgs.Args())

	_, err = ParseSSHArgs([]string{"-l", "-llogin", "host", "-p", "port", "command", "-l", "arg2"})

	require.Error(t, err)
}

func TestParseOpts(t *testing.T) {
	t.Parallel()

	opts, leftoverArgs, err := ParseOpts([]string{"-l", "login", "--use-eice", "host"})
	require.NoError(t, err)
	assert.True(t, opts.useEICE)
	assert.Equal(t, []string{"-l", "login", "host"}, leftoverArgs)

	opts, leftoverArgs, err = ParseOpts([]string{"-l", "login", "--eice-id", "eice-070594c0adf9e0f56", "host"})
	require.NoError(t, err)
	assert.True(t, opts.useEICE)
	assert.Equal(t, "eice-070594c0adf9e0f56", opts.eiceID)
	assert.Equal(t, []string{"-l", "login", "host"}, leftoverArgs)

	_, _, err = ParseOpts([]string{"-l", "login", "--eice-id"})
	require.Error(t, err)

	_, _, err = ParseOpts([]string{"-l", "login", "--destination-type", "host"})
	require.Error(t, err)

	opts, leftoverArgs, err = ParseOpts([]string{"-l", "login", "--use-eice", "host", "--", "command", "arg1", "arg2"})
	require.NoError(t, err)
	assert.True(t, opts.useEICE)
	assert.Equal(t, []string{"-l", "login", "host", "--", "command", "arg1", "arg2"}, leftoverArgs)

	opts, leftoverArgs, err = ParseOpts([]string{"-l", "login", "host", "--use-eice", "command", "arg1", "arg2"})
	require.NoError(t, err)
	assert.True(t, opts.useEICE)
	assert.Equal(t, []string{"-l", "login", "host", "command", "arg1", "arg2"}, leftoverArgs)

	opts, leftoverArgs, err = ParseOpts([]string{"host", "--use-eice", "command", "--", "arg1", "arg2", "--eice-id"})
	require.NoError(t, err)
	assert.True(t, opts.useEICE)
	assert.Equal(t, []string{"host", "command", "--", "arg1", "arg2", "--eice-id"}, leftoverArgs)

	opts, leftoverArgs, err = ParseOpts([]string{"host", "--use-eice", "--", "command", "arg1", "arg2", "--eice-id"})
	require.NoError(t, err)
	assert.True(t, opts.useEICE)
	assert.Equal(t, []string{"host", "--", "command", "arg1", "arg2", "--eice-id"}, leftoverArgs)

	// This should be an error, but it's not - long option between short option and its argument
	opts, leftoverArgs, err = ParseOpts([]string{"-l", "--use-eice", "login", "host"})
	require.NoError(t, err)
	assert.True(t, opts.useEICE)
	assert.Equal(t, []string{"-l", "login", "host"}, leftoverArgs)
}
