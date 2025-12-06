package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSSHDestination(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		destination string
		wantLogin   string
		wantHost    string
		wantPort    string
	}{
		// Basic formats
		"host only": {
			destination: "myhost",
			wantHost:    "myhost",
		},
		"user@host": {
			destination: "admin@myhost",
			wantLogin:   "admin",
			wantHost:    "myhost",
		},
		// Instance ID
		"instance id": {
			destination: "i-1234567890abcdef0",
			wantHost:    "i-1234567890abcdef0",
		},
		"user@instance id": {
			destination: "ec2-user@i-1234567890abcdef0",
			wantLogin:   "ec2-user",
			wantHost:    "i-1234567890abcdef0",
		},
		// Private DNS name
		"private dns": {
			destination: "ip-10-0-0-1.ec2.internal",
			wantHost:    "ip-10-0-0-1.ec2.internal",
		},
		"user@private dns": {
			destination: "ec2-user@ip-10-0-0-1.ec2.internal",
			wantLogin:   "ec2-user",
			wantHost:    "ip-10-0-0-1.ec2.internal",
		},
		// SSH URLs
		"ssh url host only": {
			destination: "ssh://myhost",
			wantHost:    "myhost",
		},
		"ssh url with user": {
			destination: "ssh://admin@myhost",
			wantLogin:   "admin",
			wantHost:    "myhost",
		},
		"ssh url with port": {
			destination: "ssh://myhost:2222",
			wantHost:    "myhost",
			wantPort:    "2222",
		},
		"ssh url full": {
			destination: "ssh://admin@myhost:2222",
			wantLogin:   "admin",
			wantHost:    "myhost",
			wantPort:    "2222",
		},
		// IPv6 addresses in SSH URL
		"ssh url ipv6 bracketed": {
			destination: "ssh://[::1]",
			wantHost:    "::1",
		},
		"ssh url ipv6 with port": {
			destination: "ssh://[::1]:2222",
			wantHost:    "::1",
			wantPort:    "2222",
		},
		"ssh url ipv6 with user and port": {
			destination: "ssh://admin@[::1]:2222",
			wantLogin:   "admin",
			wantHost:    "::1",
			wantPort:    "2222",
		},
		"ssh url ipv6 full address": {
			destination: "ssh://[2001:db8::1]:22",
			wantHost:    "2001:db8::1",
			wantPort:    "22",
		},
		// Email-like user (multiple @ signs)
		"email-like user": {
			destination: "user@domain.com@host",
			wantLogin:   "user@domain.com",
			wantHost:    "host",
		},
		// IP addresses
		"ipv4 address": {
			destination: "192.168.1.1",
			wantHost:    "192.168.1.1",
		},
		"user@ipv4": {
			destination: "admin@10.0.0.1",
			wantLogin:   "admin",
			wantHost:    "10.0.0.1",
		},
		// Empty string
		"empty string": {
			destination: "",
			wantHost:    "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			login, host, port := ParseSSHDestination(tc.destination)

			assert.Equal(t, tc.wantLogin, login, "login")
			assert.Equal(t, tc.wantHost, host, "host")
			assert.Equal(t, tc.wantPort, port, "port")
		})
	}
}

func TestParseSFTPDestination(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		destination string
		wantLogin   string
		wantHost    string
		wantPort    string
		wantPath    string
	}{
		// Non-URL formats
		"host only": {
			destination: "myhost",
			wantHost:    "myhost",
		},
		"host:path": {
			destination: "myhost:/home/user",
			wantHost:    "myhost",
			wantPath:    "/home/user",
		},
		"user@host:path": {
			destination: "admin@myhost:/home/user",
			wantLogin:   "admin",
			wantHost:    "myhost",
			wantPath:    "/home/user",
		},
		"host:relative path": {
			destination: "myhost:file.txt",
			wantHost:    "myhost",
			wantPath:    "file.txt",
		},
		// SFTP URLs
		"sftp url host only": {
			destination: "sftp://myhost",
			wantHost:    "myhost",
		},
		"sftp url with path": {
			destination: "sftp://myhost/home/user",
			wantHost:    "myhost",
			wantPath:    "home/user",
		},
		"sftp url with port": {
			destination: "sftp://myhost:2222/path",
			wantHost:    "myhost",
			wantPort:    "2222",
			wantPath:    "path",
		},
		"sftp url full": {
			destination: "sftp://admin@myhost:2222/home/user",
			wantLogin:   "admin",
			wantHost:    "myhost",
			wantPort:    "2222",
			wantPath:    "home/user",
		},
		"sftp url no path": {
			destination: "sftp://admin@myhost:2222",
			wantLogin:   "admin",
			wantHost:    "myhost",
			wantPort:    "2222",
		},
		// IPv6 non-URL
		"ipv6 with path": {
			destination: "[::1]:/home/user",
			wantHost:    "::1",
			wantPath:    "/home/user",
		},
		"user@ipv6 with path": {
			destination: "admin@[::1]:/home/user",
			wantLogin:   "admin",
			wantHost:    "::1",
			wantPath:    "/home/user",
		},
		// SFTP URL with IPv6
		"sftp ipv6 url": {
			destination: "sftp://[::1]:2222/path",
			wantHost:    "::1",
			wantPort:    "2222",
			wantPath:    "path",
		},
		"sftp ipv6 url with user": {
			destination: "sftp://admin@[2001:db8::1]:22/data",
			wantLogin:   "admin",
			wantHost:    "2001:db8::1",
			wantPort:    "22",
			wantPath:    "data",
		},
		// Instance ID
		"instance id with path": {
			destination: "i-1234567890abcdef0:/home/ec2-user",
			wantHost:    "i-1234567890abcdef0",
			wantPath:    "/home/ec2-user",
		},
		// Empty string
		"empty string": {
			destination: "",
			wantHost:    "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			login, host, port, path := ParseSFTPDestination(tc.destination)

			assert.Equal(t, tc.wantLogin, login, "login")
			assert.Equal(t, tc.wantHost, host, "host")
			assert.Equal(t, tc.wantPort, port, "port")
			assert.Equal(t, tc.wantPath, path, "path")
		})
	}
}

func TestStripIPv6Brackets(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  string
	}{
		"bracketed ipv6":        {input: "[::1]", want: "::1"},
		"bracketed full ipv6":   {input: "[2001:db8::1]", want: "2001:db8::1"},
		"unbracketed ipv6":      {input: "::1", want: "::1"},
		"ipv4 unchanged":        {input: "192.168.1.1", want: "192.168.1.1"},
		"hostname unchanged":    {input: "myhost", want: "myhost"},
		"empty string":          {input: "", want: ""},
		"only open bracket":     {input: "[::1", want: "[::1"},
		"only close bracket":    {input: "::1]", want: "::1]"},
		"brackets with content": {input: "[abc]", want: "abc"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, StripIPv6Brackets(tc.input))
		})
	}
}
