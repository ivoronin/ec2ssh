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
	assert.Equal(t, "fec1::1", host)
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

func TestNewOptions(t *testing.T) {
	t.Parallel()

	parsedArgs := ParsedArgs{
		Options: map[string]string{
			"-l":         "login",
			"--use-eice": "true",
		},
		Destination:     "host",
		CommandWithArgs: []string{"command"},
		SSHArgs:         []string{"-t"},
	}

	session, err := NewOptions(parsedArgs)

	require.NoError(t, err)
	assert.Equal(t, "host", session.Destination)
	assert.Equal(t, "login", session.Login)
	assert.True(t, session.UseEICE)
}
