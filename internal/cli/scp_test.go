package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSCPOperand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		operand    string
		wantLogin  string
		wantHost   string
		wantPath   string
		wantRemote bool
	}{
		// Local paths - absolute
		"absolute path": {
			operand:  "/home/user/file.txt",
			wantPath: "/home/user/file.txt",
		},
		"root path": {
			operand:  "/",
			wantPath: "/",
		},
		// Local paths - relative
		"relative path dot": {
			operand:  "./file.txt",
			wantPath: "./file.txt",
		},
		"relative path dotdot": {
			operand:  "../file.txt",
			wantPath: "../file.txt",
		},
		"current dir": {
			operand:  ".",
			wantPath: ".",
		},
		"simple filename": {
			operand:  "file.txt",
			wantPath: "file.txt",
		},
		// Local paths - with colons
		"path with colon after slash": {
			operand:  "/home/user/file:with:colons.txt",
			wantPath: "/home/user/file:with:colons.txt",
		},
		"leading colon is local": {
			operand:  ":filename",
			wantPath: ":filename",
		},
		// Local paths - tilde
		"tilde path": {
			operand:  "~/file.txt",
			wantPath: "~/file.txt",
		},
		// Remote operands - basic
		"remote host:path": {
			operand:    "host:/path/file",
			wantHost:   "host",
			wantPath:   "/path/file",
			wantRemote: true,
		},
		"remote user@host:path": {
			operand:    "admin@host:/path/file",
			wantLogin:  "admin",
			wantHost:   "host",
			wantPath:   "/path/file",
			wantRemote: true,
		},
		"remote host:relative": {
			operand:    "host:file.txt",
			wantHost:   "host",
			wantPath:   "file.txt",
			wantRemote: true,
		},
		"remote host empty path": {
			operand:    "host:",
			wantHost:   "host",
			wantPath:   "",
			wantRemote: true,
		},
		// Remote operands - IPv6
		"ipv6 remote": {
			operand:    "[::1]:/path",
			wantHost:   "::1",
			wantPath:   "/path",
			wantRemote: true,
		},
		"ipv6 with user": {
			operand:    "admin@[::1]:/path",
			wantLogin:  "admin",
			wantHost:   "::1",
			wantPath:   "/path",
			wantRemote: true,
		},
		"ipv6 full address": {
			operand:    "[2001:db8::1]:/data",
			wantHost:   "2001:db8::1",
			wantPath:   "/data",
			wantRemote: true,
		},
		// Remote operands - instance IDs
		"instance id remote": {
			operand:    "i-1234567890abcdef0:/path",
			wantHost:   "i-1234567890abcdef0",
			wantPath:   "/path",
			wantRemote: true,
		},
		"user@instance id remote": {
			operand:    "ec2-user@i-1234567890abcdef0:/home/ec2-user",
			wantLogin:  "ec2-user",
			wantHost:   "i-1234567890abcdef0",
			wantPath:   "/home/ec2-user",
			wantRemote: true,
		},
		// Edge cases
		"empty string": {
			operand:  "",
			wantPath: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := ParseSCPOperand(tc.operand)

			assert.Equal(t, tc.wantLogin, result.Login, "login")
			assert.Equal(t, tc.wantHost, result.Host, "host")
			assert.Equal(t, tc.wantPath, result.Path, "path")
			assert.Equal(t, tc.wantRemote, result.IsRemote, "isRemote")
		})
	}
}

func TestParseSCPOperands(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		operands       []string
		wantLogin      string
		wantHost       string
		wantRemotePath string
		wantLocalPath  string
		wantIsUpload   bool
		wantErr        bool
		errContains    string
	}{
		// Valid upload
		"upload local to remote": {
			operands:       []string{"/local/file.txt", "host:/remote/file.txt"},
			wantHost:       "host",
			wantRemotePath: "/remote/file.txt",
			wantLocalPath:  "/local/file.txt",
			wantIsUpload:   true,
		},
		"upload with user": {
			operands:       []string{"file.txt", "admin@host:/path"},
			wantLogin:      "admin",
			wantHost:       "host",
			wantRemotePath: "/path",
			wantLocalPath:  "file.txt",
			wantIsUpload:   true,
		},
		"upload relative to remote": {
			operands:       []string{"./local.txt", "host:remote.txt"},
			wantHost:       "host",
			wantRemotePath: "remote.txt",
			wantLocalPath:  "./local.txt",
			wantIsUpload:   true,
		},
		// Valid download
		"download remote to local": {
			operands:       []string{"host:/remote/file.txt", "/local/"},
			wantHost:       "host",
			wantRemotePath: "/remote/file.txt",
			wantLocalPath:  "/local/",
			wantIsUpload:   false,
		},
		"download with user": {
			operands:       []string{"admin@host:/path", "."},
			wantLogin:      "admin",
			wantHost:       "host",
			wantRemotePath: "/path",
			wantLocalPath:  ".",
			wantIsUpload:   false,
		},
		"download to current dir": {
			operands:       []string{"host:/file.txt", "."},
			wantHost:       "host",
			wantRemotePath: "/file.txt",
			wantLocalPath:  ".",
			wantIsUpload:   false,
		},
		// IPv6 remote
		"upload to ipv6": {
			operands:       []string{"local.txt", "[::1]:/remote.txt"},
			wantHost:       "::1",
			wantRemotePath: "/remote.txt",
			wantLocalPath:  "local.txt",
			wantIsUpload:   true,
		},
		"download from ipv6": {
			operands:       []string{"admin@[2001:db8::1]:/data", "/local"},
			wantLogin:      "admin",
			wantHost:       "2001:db8::1",
			wantRemotePath: "/data",
			wantLocalPath:  "/local",
			wantIsUpload:   false,
		},
		// Error cases
		"wrong operand count - zero": {
			operands:    []string{},
			wantErr:     true,
			errContains: "exactly 2 operands",
		},
		"wrong operand count - one": {
			operands:    []string{"host:/path"},
			wantErr:     true,
			errContains: "exactly 2 operands",
		},
		"wrong operand count - three": {
			operands:    []string{"a", "b", "c"},
			wantErr:     true,
			errContains: "exactly 2 operands",
		},
		"both local": {
			operands:    []string{"/local/a", "/local/b"},
			wantErr:     true,
			errContains: "no remote operand",
		},
		"both remote": {
			operands:    []string{"host1:/path", "host2:/path"},
			wantErr:     true,
			errContains: "multiple remote operands",
		},
		"empty remote path": {
			operands:    []string{"file.txt", "host:"},
			wantErr:     true,
			errContains: "remote path cannot be empty",
		},
		"empty host - leading colon": {
			operands:    []string{"file.txt", ":/path"},
			wantErr:     true,
			errContains: "no remote operand", // Leading colon makes it local
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseSCPOperands(tc.operands)

			if tc.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrSCP)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantLogin, result.Login, "login")
			assert.Equal(t, tc.wantHost, result.Host, "host")
			assert.Equal(t, tc.wantRemotePath, result.RemotePath, "remotePath")
			assert.Equal(t, tc.wantLocalPath, result.LocalPath, "localPath")
			assert.Equal(t, tc.wantIsUpload, result.IsUpload, "isUpload")
		})
	}
}

func TestFindColonSeparator(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  int
	}{
		// Local paths (return -1)
		"empty string":            {input: "", want: -1},
		"leading colon":           {input: ":filename", want: -1},
		"slash before colon":      {input: "/path/file:txt", want: -1},
		"relative slash":          {input: "./file:txt", want: -1},
		"just filename":           {input: "filename", want: -1},
		"tilde path":              {input: "~/file.txt", want: -1},
		"no colon":                {input: "hostname", want: -1},
		"ipv6 no separator":       {input: "[::1]", want: -1},
		"ipv6 brackets only":      {input: "[2001:db8::1]", want: -1},
		"dotdot path with colon":  {input: "../path/file:txt", want: -1},
		"local path starts slash": {input: "/home/user/archive:backup.tar", want: -1},

		// Remote paths (return index)
		"simple host:path":       {input: "host:path", want: 4},
		"user@host:path":         {input: "user@host:path", want: 9},
		"ipv6 with path":         {input: "[::1]:path", want: 5},
		"user@ipv6:path":         {input: "user@[::1]:path", want: 10},
		"full ipv6 with path":    {input: "[2001:db8::1]:/data", want: 13},
		"user@full ipv6:path":    {input: "admin@[2001:db8::1]:/data", want: 19},
		"host empty path":        {input: "host:", want: 4},
		"instance id":            {input: "i-abc123:/path", want: 8},
		"hostname with dots":     {input: "web.server.com:/var/www", want: 14},
		"email-like user":        {input: "user@domain.com@host:/path", want: 20},
		"port-like in path":      {input: "host:8080", want: 4}, // This is host:path, not host:port
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, findColonSeparator(tc.input))
		})
	}
}
