package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSSHDestination(t *testing.T) {
	t.Parallel()

	// URL like destination
	login, host, port := ParseSSHDestination("ssh://login@host:port")

	assert.Equal(t, "login", login)
	assert.Equal(t, "host", host)
	assert.Equal(t, "port", port)

	login, host, port = ParseSSHDestination("ssh://host:port")

	assert.Equal(t, "", login)
	assert.Equal(t, "host", host)
	assert.Equal(t, "port", port)

	login, host, port = ParseSSHDestination("ssh://host")

	assert.Equal(t, "", login)
	assert.Equal(t, "host", host)
	assert.Equal(t, "", port)

	login, host, port = ParseSSHDestination("ssh://login@domain@host:port")

	assert.Equal(t, "login@domain", login)
	assert.Equal(t, "host", host)
	assert.Equal(t, "port", port)

	login, host, port = ParseSSHDestination("ssh://login@domain@[fec1::1]:port")

	assert.Equal(t, "login@domain", login)
	assert.Equal(t, "fec1::1", host)
	assert.Equal(t, "port", port)

	login, host, port = ParseSSHDestination("ssh://login@domain@[fec1::1]")

	assert.Equal(t, "login@domain", login)
	assert.Equal(t, "fec1::1", host)
	assert.Equal(t, "", port)

	login, host, port = ParseSSHDestination("ssh://@[fec1::1]")

	assert.Equal(t, "", login)
	assert.Equal(t, "fec1::1", host)
	assert.Equal(t, "", port)

	login, host, port = ParseSSHDestination("[fec1::1]")

	assert.Equal(t, "", login)
	assert.Equal(t, "[fec1::1]", host)
	assert.Equal(t, "", port)

	// Non-URL like destination
	login, host, port = ParseSSHDestination("login@host")
	assert.Equal(t, "login", login)
	assert.Equal(t, "host", host)
	assert.Equal(t, "", port)

	login, host, port = ParseSSHDestination("host")
	assert.Equal(t, "", login)
	assert.Equal(t, "host", host)
	assert.Equal(t, "", port)

	login, host, port = ParseSSHDestination("host:port")
	assert.Equal(t, "", login)
	assert.Equal(t, "host:port", host)
	assert.Equal(t, "", port)

	login, host, port = ParseSSHDestination("login@host:port")
	assert.Equal(t, "login", login)
	assert.Equal(t, "host:port", host)
	assert.Equal(t, "", port)
}
