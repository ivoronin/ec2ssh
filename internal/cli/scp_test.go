package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindColonSeparator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  int
	}{
		// Local paths (return -1)
		{"", -1},
		{"file.txt", -1},
		{":leading-colon", -1},        // OpenSSH: leading colon is filename
		{"/path/to/file", -1},         // Slash before any colon
		{"/path/with:colon", -1},      // Slash before colon
		{"./relative/path:file", -1},  // Slash before colon
		{"path/to:file", -1},          // Slash before colon
		{"[::1]", -1},                 // IPv6 without trailing colon

		// Windows paths - treated as remote by OpenSSH (colon after drive letter)
		// Note: OpenSSH scp does NOT special-case Windows drive letters
		{"C:file.txt", 1},
		{"C:\\Users\\file.txt", 1}, // Backslash doesn't stop colon search
		{"D:\\path\\to\\file", 1},

		// Remote paths (return colon index)
		{"host:path", 4},
		{"host:", 4},
		{"user@host:path", 9},
		{"[::1]:/path", 5},
		{"user@[::1]:/path", 10},
		{"user@domain@host:path", 16},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := findColonSeparator(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseSCPOperand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantLogin  string
		wantHost   string
		wantPath   string
		wantRemote bool
	}{
		// Local paths - basic
		{
			name:       "absolute path",
			input:      "/var/log/app.log",
			wantPath:   "/var/log/app.log",
			wantRemote: false,
		},
		{
			name:       "relative path with dot",
			input:      "./local/file.txt",
			wantPath:   "./local/file.txt",
			wantRemote: false,
		},
		{
			name:       "home path",
			input:      "~/documents/file.txt",
			wantPath:   "~/documents/file.txt",
			wantRemote: false,
		},
		{
			name:       "simple relative path",
			input:      "file.txt",
			wantPath:   "file.txt",
			wantRemote: false,
		},

		// Local paths - OpenSSH edge cases
		{
			name:       "leading colon is local filename",
			input:      ":filename",
			wantPath:   ":filename",
			wantRemote: false,
		},
		{
			name:       "slash before colon is local",
			input:      "path/to:file",
			wantPath:   "path/to:file",
			wantRemote: false,
		},
		{
			name:       "absolute path with colon",
			input:      "/path/to/file:with:colons",
			wantPath:   "/path/to/file:with:colons",
			wantRemote: false,
		},
		{
			name:       "relative path with colon after slash",
			input:      "./file:with:colons",
			wantPath:   "./file:with:colons",
			wantRemote: false,
		},

		// Remote paths
		{
			name:       "instance-id with path",
			input:      "i-0123456789abcdef0:/var/log/app.log",
			wantHost:   "i-0123456789abcdef0",
			wantPath:   "/var/log/app.log",
			wantRemote: true,
		},
		{
			name:       "user@instance-id with path",
			input:      "ec2-user@i-0123456789abcdef0:/home/ec2-user/file",
			wantLogin:  "ec2-user",
			wantHost:   "i-0123456789abcdef0",
			wantPath:   "/home/ec2-user/file",
			wantRemote: true,
		},
		{
			name:       "name tag with path",
			input:      "app-server:/etc/config.yaml",
			wantHost:   "app-server",
			wantPath:   "/etc/config.yaml",
			wantRemote: true,
		},
		{
			name:       "user@name-tag with path",
			input:      "ubuntu@web-prod:/var/www/html",
			wantLogin:  "ubuntu",
			wantHost:   "web-prod",
			wantPath:   "/var/www/html",
			wantRemote: true,
		},
		{
			name:       "remote with relative path",
			input:      "i-123:~/file.txt",
			wantHost:   "i-123",
			wantPath:   "~/file.txt",
			wantRemote: true,
		},
		{
			name:       "private IP with path",
			input:      "10.0.0.5:/data/backup",
			wantHost:   "10.0.0.5",
			wantPath:   "/data/backup",
			wantRemote: true,
		},

		// IPv6 remote paths
		{
			name:       "IPv6 with path",
			input:      "[::1]:/path/file",
			wantHost:   "::1",
			wantPath:   "/path/file",
			wantRemote: true,
		},
		{
			name:       "user@IPv6 with path",
			input:      "user@[fec1::1]:/remote/path",
			wantLogin:  "user",
			wantHost:   "fec1::1",
			wantPath:   "/remote/path",
			wantRemote: true,
		},
		{
			name:       "IPv6 without colon after bracket (local)",
			input:      "[::1]",
			wantPath:   "[::1]",
			wantRemote: false,
		},

		// Edge cases
		{
			name:       "user with @ in name",
			input:      "user@domain@host:/path",
			wantLogin:  "user@domain",
			wantHost:   "host",
			wantPath:   "/path",
			wantRemote: true,
		},

		// Windows paths - OpenSSH treats these as REMOTE (drive letter looks like host)
		// Users must use forward slashes or ./ prefix to force local interpretation
		{
			name:       "Windows drive letter treated as remote",
			input:      "C:file.txt",
			wantHost:   "C",
			wantPath:   "file.txt",
			wantRemote: true,
		},
		{
			name:       "Windows path with backslash treated as remote",
			input:      "C:\\Users\\file.txt",
			wantHost:   "C",
			wantPath:   "\\Users\\file.txt",
			wantRemote: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ParseSCPOperand(tt.input)

			assert.Equal(t, tt.wantRemote, result.IsRemote, "IsRemote mismatch")
			assert.Equal(t, tt.wantLogin, result.Login, "Login mismatch")
			assert.Equal(t, tt.wantHost, result.Host, "Host mismatch")
			assert.Equal(t, tt.wantPath, result.Path, "Path mismatch")
		})
	}
}

func TestParseSCPOperands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		operands       []string
		wantLogin      string
		wantHost       string
		wantRemotePath string
		wantLocalPath  string
		wantIsUpload   bool
		wantErr        error
	}{
		// Valid download cases
		{
			name:           "download basic",
			operands:       []string{"i-123:/var/log/app.log", "./"},
			wantHost:       "i-123",
			wantRemotePath: "/var/log/app.log",
			wantLocalPath:  "./",
			wantIsUpload:   false,
		},
		{
			name:           "download with user",
			operands:       []string{"ec2-user@app-server:/home/ec2-user/data", "/tmp/"},
			wantLogin:      "ec2-user",
			wantHost:       "app-server",
			wantRemotePath: "/home/ec2-user/data",
			wantLocalPath:  "/tmp/",
			wantIsUpload:   false,
		},
		{
			name:           "download to current dir",
			operands:       []string{"i-abc123:/etc/config", "."},
			wantHost:       "i-abc123",
			wantRemotePath: "/etc/config",
			wantLocalPath:  ".",
			wantIsUpload:   false,
		},

		// Valid upload cases
		{
			name:           "upload basic",
			operands:       []string{"./config.json", "i-123:/etc/app/"},
			wantHost:       "i-123",
			wantRemotePath: "/etc/app/",
			wantLocalPath:  "./config.json",
			wantIsUpload:   true,
		},
		{
			name:           "upload with user",
			operands:       []string{"/local/file.txt", "ubuntu@web-prod:/var/www/"},
			wantLogin:      "ubuntu",
			wantHost:       "web-prod",
			wantRemotePath: "/var/www/",
			wantLocalPath:  "/local/file.txt",
			wantIsUpload:   true,
		},
		{
			name:           "upload relative local path",
			operands:       []string{"data.tar.gz", "i-123:/backup/"},
			wantHost:       "i-123",
			wantRemotePath: "/backup/",
			wantLocalPath:  "data.tar.gz",
			wantIsUpload:   true,
		},

		// IPv6 cases
		{
			name:           "download from IPv6",
			operands:       []string{"user@[::1]:/path", "/local"},
			wantLogin:      "user",
			wantHost:       "::1",
			wantRemotePath: "/path",
			wantLocalPath:  "/local",
			wantIsUpload:   false,
		},

		// Error cases
		{
			name:     "no operands",
			operands: []string{},
			wantErr:  ErrSCP,
		},
		{
			name:     "one operand",
			operands: []string{"i-123:/path"},
			wantErr:  ErrSCP,
		},
		{
			name:     "three operands",
			operands: []string{"a", "b", "c"},
			wantErr:  ErrSCP,
		},
		{
			name:     "both local",
			operands: []string{"/path1", "/path2"},
			wantErr:  ErrSCP,
		},
		{
			name:     "both local relative",
			operands: []string{"file1.txt", "file2.txt"},
			wantErr:  ErrSCP,
		},
		{
			name:     "both remote",
			operands: []string{"i-123:/a", "i-456:/b"},
			wantErr:  ErrSCP,
		},
		{
			name:     "empty path after colon",
			operands: []string{"i-123:", "/local"},
			wantErr:  ErrSCP,
		},
		{
			name:     "leading colon treated as local (no remote)",
			operands: []string{":/path", "/local"},
			wantErr:  ErrSCP, // OpenSSH: leading colon is filename
		},

		// Edge cases - local files with colons (OpenSSH-compatible)
		{
			name:           "local file with colon after slash",
			operands:       []string{"path/to:file", "i-123:/dest"},
			wantHost:       "i-123",
			wantRemotePath: "/dest",
			wantLocalPath:  "path/to:file",
			wantIsUpload:   true,
		},
		{
			name:           "local file with dot prefix and colon",
			operands:       []string{"./file:with:colons", "i-123:/dest"},
			wantHost:       "i-123",
			wantRemotePath: "/dest",
			wantLocalPath:  "./file:with:colons",
			wantIsUpload:   true,
		},
		{
			name:           "leading colon is local filename",
			operands:       []string{":oddname", "i-123:/dest"},
			wantHost:       "i-123",
			wantRemotePath: "/dest",
			wantLocalPath:  ":oddname",
			wantIsUpload:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseSCPOperands(tt.operands)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantLogin, result.Login, "Login mismatch")
			assert.Equal(t, tt.wantHost, result.Host, "Host mismatch")
			assert.Equal(t, tt.wantRemotePath, result.RemotePath, "RemotePath mismatch")
			assert.Equal(t, tt.wantLocalPath, result.LocalPath, "LocalPath mismatch")
			assert.Equal(t, tt.wantIsUpload, result.IsUpload, "IsUpload mismatch")
		})
	}
}
