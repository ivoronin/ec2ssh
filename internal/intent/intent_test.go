package intent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		binPath    string
		args       []string
		wantIntent Intent
		wantArgs   []string
	}{
		// Binary name detection
		{
			name:       "ec2ssh default",
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"host"},
			wantIntent: IntentSSH,
			wantArgs:   []string{"host"},
		},
		{
			name:       "ec2list binary",
			binPath:    "/usr/bin/ec2list",
			args:       nil,
			wantIntent: IntentList,
			wantArgs:   nil,
		},
		{
			name:       "ec2list with args",
			binPath:    "/usr/bin/ec2list",
			args:       []string{"--region", "us-west-2"},
			wantIntent: IntentList,
			wantArgs:   []string{"--region", "us-west-2"},
		},
		{
			name:       "unknown binary defaults to SSH",
			binPath:    "/usr/bin/ec2foo",
			args:       []string{"host"},
			wantIntent: IntentSSH,
			wantArgs:   []string{"host"},
		},
		{
			name:       "empty binary name defaults to SSH",
			binPath:    "ec2ssh",
			args:       []string{"host"},
			wantIntent: IntentSSH,
			wantArgs:   []string{"host"},
		},

		// Override flags
		{
			name:       "--list override",
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--list"},
			wantIntent: IntentList,
			wantArgs:   []string{},
		},
		{
			name:       "--list with additional args",
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--list", "--region", "us-east-1"},
			wantIntent: IntentList,
			wantArgs:   []string{"--region", "us-east-1"},
		},
		{
			name:       "--ssh explicit",
			binPath:    "/usr/bin/ec2list",
			args:       []string{"--ssh", "host"},
			wantIntent: IntentSSH,
			wantArgs:   []string{"host"},
		},
		{
			name:       "--help long form",
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--help"},
			wantIntent: IntentHelp,
			wantArgs:   []string{},
		},
		{
			name:       "-h short form",
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"-h"},
			wantIntent: IntentHelp,
			wantArgs:   []string{},
		},
		{
			name:       "--wscat tunnel",
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--wscat"},
			wantIntent: IntentTunnel,
			wantArgs:   []string{},
		},

		// Override wins over binary name (silently)
		{
			name:       "--ssh overrides ec2list binary",
			binPath:    "/usr/bin/ec2list",
			args:       []string{"--ssh", "host"},
			wantIntent: IntentSSH,
			wantArgs:   []string{"host"},
		},
		{
			name:       "--list overrides ec2ssh binary",
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--list", "--region", "eu-west-1"},
			wantIntent: IntentList,
			wantArgs:   []string{"--region", "eu-west-1"},
		},

		// Edge cases
		{
			name:       "no args with ec2ssh",
			binPath:    "/usr/bin/ec2ssh",
			args:       nil,
			wantIntent: IntentSSH,
			wantArgs:   nil,
		},
		{
			name:       "no args with ec2list",
			binPath:    "/usr/bin/ec2list",
			args:       []string{},
			wantIntent: IntentList,
			wantArgs:   []string{},
		},
		{
			name:       "non-intent flag is not consumed",
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--region", "us-west-2", "host"},
			wantIntent: IntentSSH,
			wantArgs:   []string{"--region", "us-west-2", "host"},
		},
		{
			name:       "--list in non-first position is not an override",
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"host", "--list"},
			wantIntent: IntentSSH,
			wantArgs:   []string{"host", "--list"},
		},
		// SFTP intent
		{
			name:       "--sftp override",
			binPath:    "/usr/bin/ec2ssh",
			args:       []string{"--sftp", "user@host:/path"},
			wantIntent: IntentSFTP,
			wantArgs:   []string{"user@host:/path"},
		},
		{
			name:       "ec2sftp binary",
			binPath:    "/usr/bin/ec2sftp",
			args:       []string{"user@host"},
			wantIntent: IntentSFTP,
			wantArgs:   []string{"user@host"},
		},
		{
			name:       "--sftp overrides ec2list binary",
			binPath:    "/usr/bin/ec2list",
			args:       []string{"--sftp", "host"},
			wantIntent: IntentSFTP,
			wantArgs:   []string{"host"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotIntent, gotArgs := Resolve(tt.binPath, tt.args)

			assert.Equal(t, tt.wantIntent, gotIntent, "intent mismatch")
			assert.Equal(t, tt.wantArgs, gotArgs, "args mismatch")
		})
	}
}

func TestIntent_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		intent Intent
		want   string
	}{
		{IntentSSH, "ssh"},
		{IntentList, "list"},
		{IntentHelp, "help"},
		{IntentTunnel, "tunnel"},
		{IntentSFTP, "sftp"},
		{Intent(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.intent.String())
		})
	}
}
