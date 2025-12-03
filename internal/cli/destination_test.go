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

func TestParseSFTPDestination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantLogin string
		wantHost  string
		wantPort  string
		wantPath  string
	}{
		// Basic forms
		{
			name:     "host only",
			input:    "host",
			wantHost: "host",
		},
		{
			name:      "user@host",
			input:     "user@host",
			wantLogin: "user",
			wantHost:  "host",
		},
		{
			name:      "user@host:path",
			input:     "user@host:/remote/path",
			wantLogin: "user",
			wantHost:  "host",
			wantPath:  "/remote/path",
		},
		{
			name:     "host:path without user",
			input:    "host:/remote/path",
			wantHost: "host",
			wantPath: "/remote/path",
		},

		// SFTP URL forms
		{
			name:      "sftp://user@host",
			input:     "sftp://user@host",
			wantLogin: "user",
			wantHost:  "host",
		},
		{
			name:      "sftp://user@host:port",
			input:     "sftp://user@host:22",
			wantLogin: "user",
			wantHost:  "host",
			wantPort:  "22",
		},
		{
			name:      "sftp://user@host/path",
			input:     "sftp://user@host/remote/path",
			wantLogin: "user",
			wantHost:  "host",
			wantPath:  "remote/path",
		},
		{
			name:      "sftp://user@host:port/path",
			input:     "sftp://user@host:2222/remote/path",
			wantLogin: "user",
			wantHost:  "host",
			wantPort:  "2222",
			wantPath:  "remote/path",
		},
		{
			name:     "sftp://host only",
			input:    "sftp://host",
			wantHost: "host",
		},

		// IPv6 forms (brackets are stripped from host)
		{
			name:      "user@[ipv6]:path",
			input:     "user@[fec1::1]:/path",
			wantLogin: "user",
			wantHost:  "fec1::1",
			wantPath:  "/path",
		},
		{
			name:      "sftp://user@[ipv6]:port/path",
			input:     "sftp://user@[fec1::1]:22/path",
			wantLogin: "user",
			wantHost:  "fec1::1",
			wantPort:  "22",
			wantPath:  "path",
		},

		// Edge cases
		{
			name:      "user with @ in name",
			input:     "user@domain@host:/path",
			wantLogin: "user@domain",
			wantHost:  "host",
			wantPath:  "/path",
		},
		{
			name:     "empty path after colon",
			input:    "host:",
			wantHost: "host",
			wantPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			login, host, port, path := ParseSFTPDestination(tt.input)

			assert.Equal(t, tt.wantLogin, login, "login mismatch")
			assert.Equal(t, tt.wantHost, host, "host mismatch")
			assert.Equal(t, tt.wantPort, port, "port mismatch")
			assert.Equal(t, tt.wantPath, path, "path mismatch")
		})
	}
}
