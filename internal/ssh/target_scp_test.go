package ssh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLocalPath(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  bool
	}{
		// Local paths
		"absolute path":    {input: "/local/path", want: true},
		"relative current": {input: "./relative", want: true},
		"relative parent":  {input: "../parent", want: true},
		"just dot":         {input: ".", want: true},
		"just dotdot":      {input: "..", want: true},
		"simple filename":  {input: "file.txt", want: true},
		"leading colon":    {input: ":file", want: true},
		"path with colon":  {input: "/path/to:file", want: true},

		// Remote targets
		"remote host path":             {input: "host:path", want: false},
		"remote user at host path":     {input: "user@host:path", want: false},
		"remote ipv6 with path":        {input: "[::1]:/path", want: false},
		"remote user at ipv6 path":     {input: "user@[::1]:/path", want: false},
		"ssh url":                      {input: "ssh://host", want: false},
		"scp url":                      {input: "scp://host/path", want: false},
		"sftp url":                     {input: "sftp://host", want: false},
		"brackets mid-string not ipv6": {input: "ho[st:pa]th", want: false}, // OpenSSH: brackets only matter at start or after @
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := IsLocalPath(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}
