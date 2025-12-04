package main

import (
	"bytes"
	"errors"
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

func TestRunner_Tunnel_Success(t *testing.T) {
	t.Parallel()

	var calledWithURI string
	stderr := &bytes.Buffer{}

	runner := &Runner{
		Args: []string{"ec2ssh", "--wscat"},
		Getenv: func(key string) string {
			if key == "EC2SSH_TUNNEL_URI" {
				return "wss://test.tunnel.uri"
			}
			return ""
		},
		Stderr: stderr,
		TunnelRunner: func(uri string) error {
			calledWithURI = uri
			return nil
		},
	}

	exitCode := runner.Run()

	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "wss://test.tunnel.uri", calledWithURI)
	assert.Empty(t, stderr.String())
}

func TestRunner_Tunnel_MissingEnvVar(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}

	runner := &Runner{
		Args: []string{"ec2ssh", "--wscat"},
		Getenv: func(key string) string {
			return "" // No environment variable set
		},
		Stderr: stderr,
	}

	exitCode := runner.Run()

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, stderr.String(), "EC2SSH_TUNNEL_URI environment variable not set")
}

func TestRunner_Tunnel_Error(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}

	runner := &Runner{
		Args: []string{"ec2ssh", "--wscat"},
		Getenv: func(key string) string {
			if key == "EC2SSH_TUNNEL_URI" {
				return "wss://test.uri"
			}
			return ""
		},
		Stderr: stderr,
		TunnelRunner: func(uri string) error {
			return errors.New("tunnel connection failed")
		},
	}

	exitCode := runner.Run()

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, stderr.String(), "tunnel connection failed")
}

func TestRunner_SSH_UsageError(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}

	runner := &Runner{
		Args:   []string{"ec2ssh"}, // No destination
		Getenv: func(key string) string { return "" },
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
		Getenv: func(key string) string { return "" },
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
		Getenv: func(key string) string { return "" },
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
				Getenv: func(key string) string { return "" },
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
				Getenv: func(key string) string { return "" },
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
		Getenv: func(key string) string { return "" },
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
	assert.NotNil(t, runner.Getenv)
	assert.NotNil(t, runner.Stderr)
	assert.NotNil(t, runner.TunnelRunner)
}
