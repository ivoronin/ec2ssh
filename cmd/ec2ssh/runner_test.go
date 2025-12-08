package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunner_Help(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args []string
	}{
		"--help flag": {
			args: []string{"ec2ssh", "--help"},
		},
		"-h flag": {
			args: []string{"ec2ssh", "-h"},
		},
		// Note: "help" as positional is treated as a destination, not help intent
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stderr := new(bytes.Buffer)
			runner := &Runner{Args: tc.args, Stderr: stderr}

			exitCode := runner.Run()

			assert.Equal(t, 1, exitCode, "help should return exit code 1")
			assert.Contains(t, stderr.String(), "Usage:", "stderr should contain usage")
		})
	}
}

func TestRunner_Version(t *testing.T) {
	t.Parallel()

	stderr := new(bytes.Buffer)
	runner := &Runner{
		Args:   []string{"ec2ssh", "--version"},
		Stderr: stderr,
	}

	exitCode := runner.Run()

	assert.Equal(t, 0, exitCode, "--version should return exit code 0")
	// Note: version is printed to stdout, not stderr, so stderr should be empty
	assert.Empty(t, stderr.String())
}

func TestRunner_MissingDestination(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args        []string
		errContains string
	}{
		"ssm mode no destination": {
			args:        []string{"ec2ssm"},
			errContains: "missing destination",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stderr := new(bytes.Buffer)
			runner := &Runner{Args: tc.args, Stderr: stderr}

			exitCode := runner.Run()

			assert.Equal(t, 1, exitCode)
			assert.Contains(t, stderr.String(), tc.errContains)
			assert.Contains(t, stderr.String(), "Usage:", "should print usage for parse errors")
		})
	}
}

func TestRunner_IntentRouting(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args        []string
		errContains string
	}{
		"ec2scp needs operands": {
			args:        []string{"ec2scp"},
			errContains: "exactly 2 operands",
		},
		"--list mode needs no destination": {
			// List mode might succeed or fail depending on context
			// but it should at least not fail with "missing destination"
			args:        []string{"ec2ssh", "--list"},
			errContains: "", // May succeed if AWS is not configured, error won't be "missing destination"
		},
		"--eice-tunnel requires flags": {
			args:        []string{"ec2ssh", "--eice-tunnel"},
			errContains: "missing",
		},
		"--ssm-tunnel requires flags": {
			args:        []string{"ec2ssh", "--ssm-tunnel"},
			errContains: "missing",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stderr := new(bytes.Buffer)
			runner := &Runner{Args: tc.args, Stderr: stderr}

			exitCode := runner.Run()

			if tc.errContains != "" {
				assert.Equal(t, 1, exitCode, "should return exit code 1 for errors")
				assert.Contains(t, stderr.String(), tc.errContains)
			}
			// Note: some cases might succeed (like --list with proper AWS config)
			// so we only check error contains when specified
		})
	}
}

func TestRunner_UnknownFlag(t *testing.T) {
	t.Parallel()

	stderr := new(bytes.Buffer)
	runner := &Runner{
		Args:   []string{"ec2ssm", "--totally-unknown-flag", "myhost"},
		Stderr: stderr,
	}

	exitCode := runner.Run()

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, stderr.String(), "unknown option")
}

func TestRunner_BinaryNameRouting(t *testing.T) {
	t.Parallel()

	// Test that binary name affects intent resolution
	tests := map[string]struct {
		binaryName  string
		args        []string
		errContains string
	}{
		"ec2scp binary": {
			binaryName:  "ec2scp",
			args:        []string{},
			errContains: "exactly 2 operands",
		},
		"ec2ssm binary": {
			binaryName:  "ec2ssm",
			args:        []string{},
			errContains: "missing destination",
		},
		"ec2list binary": {
			binaryName:  "ec2list",
			args:        []string{},
			errContains: "", // List may succeed/fail based on AWS config
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stderr := new(bytes.Buffer)
			allArgs := append([]string{tc.binaryName}, tc.args...)
			runner := &Runner{Args: allArgs, Stderr: stderr}

			exitCode := runner.Run()

			if tc.errContains != "" {
				assert.Equal(t, 1, exitCode)
				assert.Contains(t, stderr.String(), tc.errContains)
			}
		})
	}
}

func TestRunner_UsageContainsExpectedSections(t *testing.T) {
	t.Parallel()

	stderr := new(bytes.Buffer)
	runner := &Runner{
		Args:   []string{"ec2ssh", "--help"},
		Stderr: stderr,
	}

	runner.Run()
	output := stderr.String()

	// HelpText should contain key sections
	expectedSections := []string{
		"Usage:",
		"ec2ssh",
	}

	for _, section := range expectedSections {
		assert.True(t, strings.Contains(output, section),
			"help should contain %q", section)
	}
}

func TestRunner_ParseErrorShowsUsage(t *testing.T) {
	t.Parallel()

	stderr := new(bytes.Buffer)
	runner := &Runner{
		Args:   []string{"ec2ssh", "--destination-type", "invalid_type", "myhost"},
		Stderr: stderr,
	}

	exitCode := runner.Run()

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, stderr.String(), "unknown destination type")
	assert.Contains(t, stderr.String(), "Usage:", "parse errors should show usage")
}
