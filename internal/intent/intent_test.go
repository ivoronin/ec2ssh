package intent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		binPath    string
		args       []string
		wantIntent Intent
		wantArgs   []string
	}{
		// Binary name mapping
		"ec2ssh binary": {
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"host"},
			wantIntent: IntentSSH,
			wantArgs:   []string{"host"},
		},
		"ec2list binary": {
			binPath:    "/usr/bin/ec2list",
			args:       []string{},
			wantIntent: IntentList,
			wantArgs:   []string{},
		},
		"ec2scp binary": {
			binPath:    "/usr/bin/ec2scp",
			args:       []string{"file", "host:/path"},
			wantIntent: IntentSCP,
			wantArgs:   []string{"file", "host:/path"},
		},
		"ec2sftp binary": {
			binPath:    "/usr/bin/ec2sftp",
			args:       []string{"host"},
			wantIntent: IntentSFTP,
			wantArgs:   []string{"host"},
		},
		"ec2ssm binary": {
			binPath:    "/usr/bin/ec2ssm",
			args:       []string{"host"},
			wantIntent: IntentSSMSession,
			wantArgs:   []string{"host"},
		},
		"unknown binary defaults to ssh": {
			binPath:    "/usr/bin/unknown",
			args:       []string{"host"},
			wantIntent: IntentSSH,
			wantArgs:   []string{"host"},
		},
		"binary with local path": {
			binPath:    "./ec2list",
			args:       []string{},
			wantIntent: IntentList,
			wantArgs:   []string{},
		},
		"binary with user path": {
			binPath:    "/home/user/.local/bin/ec2sftp",
			args:       []string{"host"},
			wantIntent: IntentSFTP,
			wantArgs:   []string{"host"},
		},

		// Flag overrides (win over binary name)
		"--ssh flag override": {
			binPath:    "/usr/bin/ec2list",
			args:       []string{"--ssh", "host"},
			wantIntent: IntentSSH,
			wantArgs:   []string{"host"},
		},
		"--list flag override": {
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--list"},
			wantIntent: IntentList,
			wantArgs:   []string{},
		},
		"--scp flag override": {
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--scp", "file", "host:/path"},
			wantIntent: IntentSCP,
			wantArgs:   []string{"file", "host:/path"},
		},
		"--sftp flag override": {
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--sftp", "host"},
			wantIntent: IntentSFTP,
			wantArgs:   []string{"host"},
		},
		"--ssm flag override": {
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--ssm", "host"},
			wantIntent: IntentSSMSession,
			wantArgs:   []string{"host"},
		},
		"--eice-tunnel flag": {
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--eice-tunnel", "--host", "10.0.0.1"},
			wantIntent: IntentEICETunnel,
			wantArgs:   []string{"--host", "10.0.0.1"},
		},
		"--ssm-tunnel flag": {
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--ssm-tunnel", "--instance-id", "i-123"},
			wantIntent: IntentSSMTunnel,
			wantArgs:   []string{"--instance-id", "i-123"},
		},

		// Help flags
		"--help flag": {
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--help"},
			wantIntent: IntentHelp,
			wantArgs:   []string{},
		},
		"-h flag": {
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"-h"},
			wantIntent: IntentHelp,
			wantArgs:   []string{},
		},
		"--help with args": {
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--help", "extra"},
			wantIntent: IntentHelp,
			wantArgs:   []string{"extra"},
		},

		// Version flag
		"--version flag": {
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--version"},
			wantIntent: IntentVersion,
			wantArgs:   []string{},
		},

		// Edge cases
		"empty args with ec2ssh": {
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{},
			wantIntent: IntentSSH,
			wantArgs:   []string{},
		},
		"empty args with ec2list": {
			binPath:    "/usr/bin/ec2list",
			args:       []string{},
			wantIntent: IntentList,
			wantArgs:   []string{},
		},
		"binary name only - no path": {
			binPath:    "ec2list",
			args:       []string{},
			wantIntent: IntentList,
			wantArgs:   []string{},
		},
		"flag not at first position": {
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"host", "--list"},
			wantIntent: IntentSSH,
			wantArgs:   []string{"host", "--list"}, // --list is not consumed
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			intent, args := Resolve(tc.binPath, tc.args)

			assert.Equal(t, tc.wantIntent, intent, "intent")
			assert.Equal(t, tc.wantArgs, args, "args")
		})
	}
}

func TestIntent_String(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		intent Intent
		want   string
	}{
		"help":        {intent: IntentHelp, want: "help"},
		"version":     {intent: IntentVersion, want: "version"},
		"ssh":         {intent: IntentSSH, want: "ssh"},
		"scp":         {intent: IntentSCP, want: "scp"},
		"sftp":        {intent: IntentSFTP, want: "sftp"},
		"eice-tunnel": {intent: IntentEICETunnel, want: "eice-tunnel"},
		"ssm":         {intent: IntentSSMSession, want: "ssm"},
		"ssm-tunnel":  {intent: IntentSSMTunnel, want: "ssm-tunnel"},
		"list":        {intent: IntentList, want: "list"},
		"unknown":     {intent: Intent(99), want: "unknown"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.intent.String())
		})
	}
}
