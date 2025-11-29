package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseArgsEmptyCommand(t *testing.T) {
	t.Parallel()

	parsed, err := ParseArgs([]string{"-l", "foo", "bar"})
	assert.NoError(t, err)
	assert.Equal(t, "bar", parsed.Destination)
	assert.Equal(t, []string{}, parsed.CommandWithArgs)
}

func TestParseArgsCommand(t *testing.T) {
	t.Parallel()

	parsed, err := ParseArgs([]string{"-l", "foo", "bar", "baz", "qux"})
	assert.NoError(t, err)
	assert.Equal(t, "bar", parsed.Destination)
	assert.Equal(t, []string{"baz", "qux"}, parsed.CommandWithArgs)
}

func TestParseArgsCommandWithDash(t *testing.T) {
	t.Parallel()

	parsed, err := ParseArgs([]string{"-l", "foo", "bar", "--", "baz", "qux"})
	assert.NoError(t, err)
	assert.Equal(t, "bar", parsed.Destination)
	assert.Equal(t, []string{"baz", "qux"}, parsed.CommandWithArgs)
}

func TestParseArgsHelp(t *testing.T) {
	t.Parallel()

	_, err := ParseArgs([]string{"--help"})
	assert.ErrorIs(t, err, ErrHelp)
}

func TestParseArgsHelpH(t *testing.T) {
	t.Parallel()

	_, err := ParseArgs([]string{"-h"})
	assert.ErrorIs(t, err, ErrHelp)
}

func TestParseArgsOptionWithMissingValue(t *testing.T) {
	t.Parallel()

	_, err := ParseArgs([]string{"-l"})
	assert.ErrorIs(t, err, ErrParse)
}

func TestParseArgsUnknownLongOption(t *testing.T) {
	t.Parallel()

	_, err := ParseArgs([]string{"--unknown", "bar"})
	assert.ErrorIs(t, err, ErrParse)
}

func TestParseArgsConsumedShortOptions(t *testing.T) {
	t.Parallel()

	parsed, err := ParseArgs([]string{"-l", "foo", "-p", "22", "-i", "/path/to/key", "bar"})
	assert.NoError(t, err)
	assert.Equal(t, "bar", parsed.Destination)
	assert.Equal(t, "foo", parsed.Options["-l"])
	assert.Equal(t, "22", parsed.Options["-p"])
	assert.Equal(t, "/path/to/key", parsed.Options["-i"])
}

func TestParseArgsConsumedLongOptions(t *testing.T) {
	t.Parallel()

	parsed, err := ParseArgs([]string{"--region", "us-west-2", "--profile", "foo", "--debug", "bar"})
	assert.NoError(t, err)
	assert.Equal(t, "bar", parsed.Destination)
	assert.Equal(t, "us-west-2", parsed.Options["--region"])
	assert.Equal(t, "foo", parsed.Options["--profile"])
	assert.Equal(t, "true", parsed.Options["--debug"])
}

func TestParseArgsConsumedLongOptionsWithEquals(t *testing.T) {
	t.Parallel()

	parsed, err := ParseArgs([]string{"--region=us-west-2", "bar"})
	assert.NoError(t, err)
	assert.Equal(t, "bar", parsed.Destination)
	assert.Equal(t, "us-west-2", parsed.Options["--region"])
}

func TestParseArgsUnconsumedShortOptions(t *testing.T) {
	t.Parallel()

	parsed, err := ParseArgs([]string{"-L", "8080:localhost:8080", "bar"})
	assert.NoError(t, err)
	assert.Equal(t, "bar", parsed.Destination)
	assert.Equal(t, []string{"-L8080:localhost:8080"}, parsed.SSHArgs)
}

func TestParseArgsUnconsumedShortFlags(t *testing.T) {
	t.Parallel()

	parsed, err := ParseArgs([]string{"-N", "bar"})
	assert.NoError(t, err)
	assert.Equal(t, "bar", parsed.Destination)
	assert.Equal(t, []string{"-N"}, parsed.SSHArgs)
}

func TestParseArgsCombinedShortOptions(t *testing.T) {
	t.Parallel()

	parsed, err := ParseArgs([]string{"-Nl", "foo", "bar"})
	assert.NoError(t, err)
	assert.Equal(t, "bar", parsed.Destination)
	assert.Equal(t, "foo", parsed.Options["-l"])
	assert.Equal(t, []string{"-N"}, parsed.SSHArgs)
}

func TestParseArgsConsumedShortOptionsWithAttachedValue(t *testing.T) {
	t.Parallel()

	parsed, err := ParseArgs([]string{"-lfoo", "bar"})
	assert.NoError(t, err)
	assert.Equal(t, "bar", parsed.Destination)
	assert.Equal(t, "foo", parsed.Options["-l"])
}

func TestParseArgsConsumedLongOptionsWithMissingValue(t *testing.T) {
	t.Parallel()

	_, err := ParseArgs([]string{"--region"})
	assert.ErrorIs(t, err, ErrParse)
}

func TestParseArgsDuplicateShortOptions(t *testing.T) {
	t.Parallel()

	parsed, err := ParseArgs([]string{"-l", "foo", "-l", "bar", "baz"})
	assert.NoError(t, err)
	assert.Equal(t, "foo", parsed.Options["-l"])
}

func TestParseArgsDuplicateLongOptions(t *testing.T) {
	t.Parallel()

	parsed, err := ParseArgs([]string{"--region", "us-west-2", "--region", "us-east-1", "bar"})
	assert.NoError(t, err)
	assert.Equal(t, "us-west-2", parsed.Options["--region"])
}

func TestParseArgsDuplicateBooleanLongOptions(t *testing.T) {
	t.Parallel()

	parsed, err := ParseArgs([]string{"--debug", "--debug", "bar"})
	assert.NoError(t, err)
	assert.Equal(t, "true", parsed.Options["--debug"])
}
