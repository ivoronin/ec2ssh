package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunner_Help(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "explicit --help flag",
			args: []string{"ec2ssh", "--help"},
		},
		{
			name: "short -h flag",
			args: []string{"ec2ssh", "-h"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stderr := &bytes.Buffer{}
			runner := &Runner{
				Args:   tt.args,
				Stderr: stderr,
			}

			exitCode := runner.Run()

			assert.Equal(t, 1, exitCode)
			assert.Contains(t, stderr.String(), "Usage:")
		})
	}
}

func TestRunner_EICETunnel_MissingArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		wantContains string
	}{
		{
			name:         "missing all required args",
			args:         []string{"ec2ssh", "--eice-tunnel"},
			wantContains: "missing required --host",
		},
		{
			name:         "missing eice-id",
			args:         []string{"ec2ssh", "--eice-tunnel", "--host", "10.0.0.1", "--port", "22"},
			wantContains: "missing required --eice-id",
		},
		{
			name:         "missing port",
			args:         []string{"ec2ssh", "--eice-tunnel", "--host", "10.0.0.1", "--eice-id", "eice-123"},
			wantContains: "missing required --port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stderr := &bytes.Buffer{}
			runner := &Runner{
				Args:   tt.args,
				Stderr: stderr,
			}

			exitCode := runner.Run()

			assert.Equal(t, 1, exitCode)
			assert.Contains(t, stderr.String(), tt.wantContains)
		})
	}
}

func TestRunner_SSMTunnel_MissingArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		wantContains string
	}{
		{
			name:         "missing instance-id",
			args:         []string{"ec2ssh", "--ssm-tunnel", "--port", "22"},
			wantContains: "missing required --instance-id",
		},
		{
			name:         "missing port",
			args:         []string{"ec2ssh", "--ssm-tunnel", "--instance-id", "i-1234567890abcdef0"},
			wantContains: "missing required --port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stderr := &bytes.Buffer{}
			runner := &Runner{
				Args:   tt.args,
				Stderr: stderr,
			}

			exitCode := runner.Run()

			assert.Equal(t, 1, exitCode)
			assert.Contains(t, stderr.String(), tt.wantContains)
		})
	}
}

func TestRunner_SSH_UsageError(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}

	runner := &Runner{
		Args:   []string{"ec2ssh"}, // No destination
		Stderr: stderr,
	}

	exitCode := runner.Run()

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, stderr.String(), "missing destination")
	assert.Contains(t, stderr.String(), "Usage:")
}

func TestRunner_SCP_UsageError(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}

	runner := &Runner{
		Args:   []string{"ec2scp"}, // No operands
		Stderr: stderr,
	}

	exitCode := runner.Run()

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, stderr.String(), "Usage:")
}

func TestRunner_SFTP_UsageError(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}

	runner := &Runner{
		Args:   []string{"ec2sftp"}, // No destination
		Stderr: stderr,
	}

	exitCode := runner.Run()

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, stderr.String(), "missing destination")
	assert.Contains(t, stderr.String(), "Usage:")
}

func TestRunner_BinaryNameRouting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		binaryName   string
		args         []string
		wantContains string
	}{
		{
			name:         "ec2list binary",
			binaryName:   "ec2list",
			args:         []string{"ec2list", "--unknown-flag"},
			wantContains: "unknown option",
		},
		{
			name:         "ec2sftp binary",
			binaryName:   "ec2sftp",
			args:         []string{"ec2sftp"},
			wantContains: "missing destination",
		},
		{
			name:         "ec2scp binary",
			binaryName:   "ec2scp",
			args:         []string{"ec2scp"},
			wantContains: "Usage:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stderr := &bytes.Buffer{}
			runner := &Runner{
				Args:   tt.args,
				Stderr: stderr,
			}

			exitCode := runner.Run()

			assert.Equal(t, 1, exitCode)
			assert.Contains(t, stderr.String(), tt.wantContains)
		})
	}
}

func TestRunner_IntentFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		wantContains string
	}{
		{
			name:         "--ssh explicit intent",
			args:         []string{"ec2ssh", "--ssh"},
			wantContains: "missing destination",
		},
		{
			name:         "--sftp explicit intent",
			args:         []string{"ec2ssh", "--sftp"},
			wantContains: "missing destination",
		},
		{
			name:         "--scp explicit intent",
			args:         []string{"ec2ssh", "--scp"},
			wantContains: "Usage:", // SCP requires 2 operands
		},
		{
			name:         "--list explicit intent",
			args:         []string{"ec2ssh", "--list", "--unknown-flag"},
			wantContains: "unknown option",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stderr := &bytes.Buffer{}
			runner := &Runner{
				Args:   tt.args,
				Stderr: stderr,
			}

			exitCode := runner.Run()

			assert.Equal(t, 1, exitCode)
			assert.Contains(t, stderr.String(), tt.wantContains)
		})
	}
}

func TestRunner_UnknownOption(t *testing.T) {
	t.Parallel()

	// Test with --list intent which has stricter arg parsing
	stderr := &bytes.Buffer{}

	runner := &Runner{
		Args:   []string{"ec2list", "--unknown-option"},
		Stderr: stderr,
	}

	exitCode := runner.Run()

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, stderr.String(), "unknown option")
	assert.Contains(t, stderr.String(), "Usage:")
}

func TestDefaultRunner(t *testing.T) {
	t.Parallel()

	runner := DefaultRunner()

	assert.NotNil(t, runner.Args)
	assert.NotNil(t, runner.Stderr)
}
