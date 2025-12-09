package ssh

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSSHTarget(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input     string
		wantLogin string
		wantHost  string
		wantPort  string
		wantStr   string
		wantErr   bool
	}{
		// Simple format
		"simple host": {
			input:    "myhost",
			wantHost: "myhost",
			wantStr:  "myhost",
		},
		"user at host": {
			input:     "admin@myhost",
			wantLogin: "admin",
			wantHost:  "myhost",
			wantStr:   "admin@myhost",
		},
		"instance id": {
			input:    "i-1234567890abcdef0",
			wantHost: "i-1234567890abcdef0",
			wantStr:  "i-1234567890abcdef0",
		},
		"user at instance id": {
			input:     "ec2-user@i-1234567890abcdef0",
			wantLogin: "ec2-user",
			wantHost:  "i-1234567890abcdef0",
			wantStr:   "ec2-user@i-1234567890abcdef0",
		},
		"fqdn": {
			input:    "host.example.com",
			wantHost: "host.example.com",
			wantStr:  "host.example.com",
		},
		"user at fqdn": {
			input:     "admin@host.example.com",
			wantLogin: "admin",
			wantHost:  "host.example.com",
			wantStr:   "admin@host.example.com",
		},
		"user at host with colon garbage": {
			input:     "user@host:garbage",
			wantLogin: "user",
			wantHost:  "host:garbage",
			wantStr:   "user@host:garbage",
		},
		"multiple at signs split at last": {
			input:     "user@host@garbage",
			wantLogin: "user@host",
			wantHost:  "garbage",
			wantStr:   "user@host@garbage",
		},
		"simple ipv6": {
			input:    "[::1]",
			wantHost: "[::1]",
			wantStr:  "[::1]",
		},
		"user at ipv6": {
			input:     "admin@[2001:db8::1]",
			wantLogin: "admin",
			wantHost:  "[2001:db8::1]",
			wantStr:   "admin@[2001:db8::1]",
		},
		"ipv6 with zone id": {
			input:    "[fe80::1%eth0]",
			wantHost: "[fe80::1%eth0]",
			wantStr:  "[fe80::1%eth0]",
		},
		"empty user at host": {
			input:    "@myhost",
			wantHost: "myhost",
			wantStr:  "myhost",
		},

		// URL format
		"ssh url host only": {
			input:    "ssh://myhost",
			wantHost: "myhost",
			wantStr:  "ssh://myhost",
		},
		"ssh url with user": {
			input:     "ssh://admin@myhost",
			wantLogin: "admin",
			wantHost:  "myhost",
			wantStr:   "ssh://admin@myhost",
		},
		"ssh url with port": {
			input:    "ssh://myhost:2222",
			wantHost: "myhost",
			wantPort: "2222",
			wantStr:  "ssh://myhost:2222",
		},
		"ssh url full": {
			input:     "ssh://admin@myhost:2222",
			wantLogin: "admin",
			wantHost:  "myhost",
			wantPort:  "2222",
			wantStr:   "ssh://admin@myhost:2222",
		},
		"ssh url ipv6 with port": {
			input:    "ssh://[::1]:22",
			wantHost: "::1", // Host() returns raw IPv6, String() adds brackets
			wantPort: "22",
			wantStr:  "ssh://[::1]:22",
		},
		"ssh url bare ipv6 no port": {
			input:    "ssh://[2001:db8::1]",
			wantHost: "2001:db8::1", // Host() returns raw IPv6, String() adds brackets
			wantStr:  "ssh://[2001:db8::1]",
		},
		"ssh url user at ipv6 with port": {
			input:     "ssh://admin@[2001:db8::1]:22",
			wantLogin: "admin",
			wantHost:  "2001:db8::1", // Host() returns raw IPv6, String() adds brackets
			wantPort:  "22",
			wantStr:   "ssh://admin@[2001:db8::1]:22",
		},

		// Error cases
		"empty string": {
			input:   "",
			wantErr: true,
		},
		"empty hostname in simple format": {
			input:   "user@",
			wantErr: true,
		},
		"ssh url empty hostname": {
			input:   "ssh://",
			wantErr: true,
		},
		"ssh url user but no hostname": {
			input:   "ssh://user@",
			wantErr: true,
		},
		"ssh url user at port but no hostname": {
			input:   "ssh://user@:22",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			target, err := NewSSHTarget(tc.input)

			if tc.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, ErrTarget), "error should wrap ErrTarget")
				return
			}

			require.NoError(t, err)
			require.NotNil(t, target)

			assert.Equal(t, tc.wantLogin, target.Login(), "login")
			assert.Equal(t, tc.wantHost, target.Host(), "host")
			assert.Equal(t, tc.wantPort, target.Port(), "port")
			assert.Equal(t, tc.wantStr, target.String(), "string")
		})
	}
}

func TestSSHTarget_SetHost(t *testing.T) {
	t.Parallel()

	t.Run("simple format", func(t *testing.T) {
		t.Parallel()

		target, err := NewSSHTarget("admin@my-instance")
		require.NoError(t, err)

		target.SetHost("10.0.0.5")

		assert.Equal(t, "10.0.0.5", target.Host())
		assert.Equal(t, "admin@10.0.0.5", target.String())
	})

	t.Run("url format preserves port", func(t *testing.T) {
		t.Parallel()

		target, err := NewSSHTarget("ssh://admin@my-instance:2222")
		require.NoError(t, err)

		target.SetHost("10.0.0.5")

		assert.Equal(t, "ssh://admin@10.0.0.5:2222", target.String())
	})
}

func TestNewSFTPTarget(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input     string
		wantLogin string
		wantHost  string
		wantPort  string
		wantPath  string
		wantStr   string
		wantErr   bool
	}{
		// Simple format
		"simple host": {
			input:    "myhost",
			wantHost: "myhost",
			wantStr:  "myhost",
		},
		"user at host": {
			input:     "admin@myhost",
			wantLogin: "admin",
			wantHost:  "myhost",
			wantStr:   "admin@myhost",
		},
		"host with path": {
			input:    "myhost:/home/user",
			wantHost: "myhost",
			wantPath: "/home/user",
			wantStr:  "myhost:/home/user",
		},
		"user at host with path": {
			input:     "admin@myhost:/data",
			wantLogin: "admin",
			wantHost:  "myhost",
			wantPath:  "/data",
			wantStr:   "admin@myhost:/data",
		},
		"ipv6 with path": {
			input:    "[::1]:/path",
			wantHost: "::1", // Host() returns raw IPv6, String() adds brackets
			wantPath: "/path",
			wantStr:  "[::1]:/path",
		},
		"bare ipv6 no path": {
			input:    "[2001:db8::1]",
			wantHost: "2001:db8::1", // Host() returns raw IPv6, String() adds brackets
			wantStr:  "[2001:db8::1]",
		},
		"user at bare ipv6 no path": {
			input:     "admin@[2001:db8::1]",
			wantLogin: "admin",
			wantHost:  "2001:db8::1", // Host() returns raw IPv6, String() adds brackets
			wantStr:   "admin@[2001:db8::1]",
		},
		"ipv6 with zone id": {
			input:    "[fe80::1%eth0]",
			wantHost: "fe80::1%eth0", // Host() returns raw IPv6, String() adds brackets
			wantStr:  "[fe80::1%eth0]",
		},
		"ipv6 zone id with path": {
			input:    "[fe80::1%eth0]:/data",
			wantHost: "fe80::1%eth0", // Host() returns raw IPv6, String() adds brackets
			wantPath: "/data",
			wantStr:  "[fe80::1%eth0]:/data",
		},
		"path with colons": {
			input:    "host:/path:with:colons",
			wantHost: "host",
			wantPath: "/path:with:colons",
			wantStr:  "host:/path:with:colons",
		},
		"empty user at host": {
			input:    "@myhost",
			wantHost: "myhost",
			wantStr:  "myhost",
		},

		// URL format
		"sftp url host only": {
			input:    "sftp://myhost",
			wantHost: "myhost",
			wantStr:  "sftp://myhost",
		},
		"sftp url with port": {
			input:    "sftp://myhost:2222",
			wantHost: "myhost",
			wantPort: "2222",
			wantStr:  "sftp://myhost:2222",
		},
		"sftp url full": {
			input:     "sftp://admin@myhost:22/home/user",
			wantLogin: "admin",
			wantHost:  "myhost",
			wantPort:  "22",
			wantPath:  "home/user",
			wantStr:   "sftp://admin@myhost:22/home/user",
		},
		"sftp url ipv6 with port": {
			input:    "sftp://[2001:db8::1]:22/data",
			wantHost: "2001:db8::1", // Host() returns raw IPv6, String() adds brackets
			wantPort: "22",
			wantPath: "data",
			wantStr:  "sftp://[2001:db8::1]:22/data",
		},
		"sftp url bare ipv6 no port": {
			input:    "sftp://[2001:db8::1]",
			wantHost: "2001:db8::1", // Host() returns raw IPv6, String() adds brackets
			wantStr:  "sftp://[2001:db8::1]",
		},
		"sftp url bare ipv6 with path": {
			input:    "sftp://[2001:db8::1]/data",
			wantHost: "2001:db8::1", // Host() returns raw IPv6, String() adds brackets
			wantPath: "data",
			wantStr:  "sftp://[2001:db8::1]/data",
		},
		"sftp url ipv6 without brackets malformed": {
			input:    "sftp://2001:db8::1",
			wantHost: "2001",   // parsed as host:port, not IPv6
			wantPort: "db8::1", // colon splits at first :
			wantStr:  "sftp://2001:db8::1",
		},

		// Error cases
		"empty string": {
			input:   "",
			wantErr: true,
		},
		"empty hostname in simple format": {
			input:   "user@",
			wantErr: true,
		},
		"sftp url empty hostname": {
			input:   "sftp://",
			wantErr: true,
		},
		"sftp url user but no hostname": {
			input:   "sftp://user@",
			wantErr: true,
		},
		"sftp url user at port but no hostname": {
			input:   "sftp://user@:22",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			target, err := NewSFTPTarget(tc.input)

			if tc.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, ErrTarget), "error should wrap ErrTarget")
				return
			}

			require.NoError(t, err)
			require.NotNil(t, target)

			assert.Equal(t, tc.wantLogin, target.Login(), "login")
			assert.Equal(t, tc.wantHost, target.Host(), "host")
			assert.Equal(t, tc.wantPort, target.Port(), "port")
			assert.Equal(t, tc.wantPath, target.Path(), "path")
			assert.Equal(t, tc.wantStr, target.String(), "string")
		})
	}
}

func TestSFTPTarget_SetHost(t *testing.T) {
	t.Parallel()

	t.Run("simple format preserves path", func(t *testing.T) {
		t.Parallel()

		target, err := NewSFTPTarget("admin@my-instance:/data")
		require.NoError(t, err)

		target.SetHost("10.0.0.5")

		assert.Equal(t, "10.0.0.5", target.Host())
		assert.Equal(t, "admin@10.0.0.5:/data", target.String())
	})

	t.Run("url format preserves port and path", func(t *testing.T) {
		t.Parallel()

		target, err := NewSFTPTarget("sftp://admin@my-instance:2222/home")
		require.NoError(t, err)

		target.SetHost("10.0.0.5")

		assert.Equal(t, "sftp://admin@10.0.0.5:2222/home", target.String())
	})
}

func TestNewSCPTarget(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input     string
		wantLogin string
		wantHost  string
		wantPort  string
		wantPath  string
		wantStr   string
		wantErr   bool
	}{
		// Simple format (colon required)
		"host with path": {
			input:    "myhost:file.txt",
			wantHost: "myhost",
			wantPath: "file.txt",
			wantStr:  "myhost:file.txt",
		},
		"user at host with path": {
			input:     "admin@myhost:/data",
			wantLogin: "admin",
			wantHost:  "myhost",
			wantPath:  "/data",
			wantStr:   "admin@myhost:/data",
		},
		"ipv6 with path": {
			input:    "[::1]:/path",
			wantHost: "::1", // Host() returns raw IPv6, String() adds brackets
			wantPath: "/path",
			wantStr:  "[::1]:/path",
		},
		"host with empty path": {
			input:    "myhost:",
			wantHost: "myhost",
			wantPath: "",
			wantStr:  "myhost:",
		},

		// URL format
		"scp url full": {
			input:     "scp://admin@myhost:22/path",
			wantLogin: "admin",
			wantHost:  "myhost",
			wantPort:  "22",
			wantPath:  "/path",
			wantStr:   "scp://admin@myhost:22/path",
		},
		"scp url host with path": {
			input:    "scp://myhost/file.txt",
			wantHost: "myhost",
			wantPath: "/file.txt",
			wantStr:  "scp://myhost/file.txt",
		},
		"ipv6 zone id with path": {
			input:    "[fe80::1%eth0]:/data",
			wantHost: "fe80::1%eth0", // Host() returns raw IPv6, String() adds brackets
			wantPath: "/data",
			wantStr:  "[fe80::1%eth0]:/data",
		},
		"path with colons": {
			input:    "host:/path:with:colons",
			wantHost: "host",
			wantPath: "/path:with:colons",
			wantStr:  "host:/path:with:colons",
		},
		"empty user at host with path": {
			input:    "@myhost:/data",
			wantHost: "myhost",
			wantPath: "/data",
			wantStr:  "myhost:/data",
		},

		// Error cases
		"missing colon in simple format": {
			input:   "myhost",
			wantErr: true,
		},
		"empty hostname in simple format": {
			input:   ":path",
			wantErr: true,
		},
		"empty hostname with user in simple format": {
			input:   "user@:/path",
			wantErr: true,
		},
		"scp url empty hostname": {
			input:   "scp://",
			wantErr: true,
		},
		"scp url user but no hostname": {
			input:   "scp://user@",
			wantErr: true,
		},
		"scp url user at port but no hostname": {
			input:   "scp://user@:22/path",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			target, err := NewSCPTarget(tc.input)

			if tc.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, ErrTarget), "error should wrap ErrTarget")
				return
			}

			require.NoError(t, err)
			require.NotNil(t, target)

			assert.Equal(t, tc.wantLogin, target.Login(), "login")
			assert.Equal(t, tc.wantHost, target.Host(), "host")
			assert.Equal(t, tc.wantPort, target.Port(), "port")
			assert.Equal(t, tc.wantPath, target.Path(), "path")
			assert.Equal(t, tc.wantStr, target.String(), "string")
		})
	}
}

func TestSCPTarget_SetHost(t *testing.T) {
	t.Parallel()

	t.Run("simple format preserves path", func(t *testing.T) {
		t.Parallel()

		target, err := NewSCPTarget("admin@my-instance:/data")
		require.NoError(t, err)

		target.SetHost("10.0.0.5")

		assert.Equal(t, "10.0.0.5", target.Host())
		assert.Equal(t, "admin@10.0.0.5:/data", target.String())
	})

	t.Run("url format preserves port and path", func(t *testing.T) {
		t.Parallel()

		target, err := NewSCPTarget("scp://admin@my-instance:22/home")
		require.NoError(t, err)

		target.SetHost("10.0.0.5")

		assert.Equal(t, "scp://admin@10.0.0.5:22/home", target.String())
	})
}
