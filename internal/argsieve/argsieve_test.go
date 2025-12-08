package argsieve

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFlags is a test struct covering all supported field types.
type testFlags struct {
	Region  string `short:"r" long:"region"`
	Profile string `short:"p" long:"profile"`
	Verbose bool   `short:"v" long:"verbose"`
	Debug   bool   `short:"d" long:"debug"`
}

// testEmbeddedBase is embedded in testEmbedded.
type testEmbeddedBase struct {
	Region string `short:"r" long:"region"`
}

// testEmbedded tests embedded struct field extraction.
type testEmbedded struct {
	testEmbeddedBase
	Profile string `short:"p" long:"profile"`
}

func TestSift(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args               []string
		passthroughWithArg []string
		wantRemaining      []string
		wantPositional     []string
		wantRegion         string
		wantProfile        string
		wantVerbose        bool
		wantDebug          bool
		wantErr            bool
	}{
		// Short flags with separate value
		"short flag with separate value": {
			args:       []string{"-r", "us-west-2"},
			wantRegion: "us-west-2",
		},
		// Short flag with attached value
		"short flag with attached value": {
			args:       []string{"-rus-west-2"},
			wantRegion: "us-west-2",
		},
		// Short bool flag
		"short bool flag": {
			args:        []string{"-v"},
			wantVerbose: true,
		},
		// Short flag chaining bools
		"short flag chaining bools": {
			args:        []string{"-vd"},
			wantVerbose: true,
			wantDebug:   true,
		},
		// Short flag chain with value at end
		"short flag chain with value at end": {
			args:        []string{"-vdrus-west-2"},
			wantVerbose: true,
			wantDebug:   true,
			wantRegion:  "us-west-2",
		},
		// Long flag with separate value
		"long flag with separate value": {
			args:       []string{"--region", "us-west-2"},
			wantRegion: "us-west-2",
		},
		// Long flag with equals value
		"long flag with equals value": {
			args:       []string{"--region=us-west-2"},
			wantRegion: "us-west-2",
		},
		// Long bool flag
		"long bool flag": {
			args:        []string{"--verbose"},
			wantVerbose: true,
		},
		// Unknown short flag passed through
		"unknown short flag passed through": {
			args:           []string{"-x", "foo"},
			wantRemaining:  []string{"-x"},
			wantPositional: []string{"foo"},
		},
		// Unknown long flag passed through
		"unknown long flag passed through": {
			args:           []string{"--unknown", "foo"},
			wantRemaining:  []string{"--unknown"},
			wantPositional: []string{"foo"},
		},
		// Passthrough flag with value
		"passthrough flag with value": {
			args:               []string{"-o", "StrictHostKeyChecking=no"},
			passthroughWithArg: []string{"-o"},
			wantRemaining:      []string{"-o", "StrictHostKeyChecking=no"},
		},
		// Passthrough flag with attached value
		"passthrough flag with attached value": {
			args:               []string{"-oStrictHostKeyChecking=no"},
			passthroughWithArg: []string{"-o"},
			wantRemaining:      []string{"-oStrictHostKeyChecking=no"},
		},
		// Passthrough long flag with value
		"passthrough long flag with value": {
			args:               []string{"--option", "value"},
			passthroughWithArg: []string{"--option"},
			wantRemaining:      []string{"--option", "value"},
		},
		// Positional only
		"positional only": {
			args:           []string{"host1", "host2"},
			wantPositional: []string{"host1", "host2"},
		},
		// Mixed flags and positional
		"mixed flags and positional": {
			args:           []string{"-r", "us-west-2", "host"},
			wantRegion:     "us-west-2",
			wantPositional: []string{"host"},
		},
		// Double dash terminator
		"double dash terminator": {
			args:           []string{"-v", "--", "-r", "us-west-2"},
			wantVerbose:    true,
			wantPositional: []string{"-r", "us-west-2"},
		},
		// Empty args
		"empty args": {
			args: []string{},
		},
		// Single dash is positional
		"single dash is positional": {
			args:           []string{"-"},
			wantPositional: []string{"-"},
		},
		// Multiple known and unknown mixed
		"multiple known and unknown mixed": {
			args:           []string{"-v", "-x", "--region", "us-east-1", "--unknown", "host"},
			wantVerbose:    true,
			wantRegion:     "us-east-1",
			wantRemaining:  []string{"-x", "--unknown"},
			wantPositional: []string{"host"},
		},
		// Missing value for short flag
		"missing value for short flag": {
			args:    []string{"-r"},
			wantErr: true,
		},
		// Missing value for long flag
		"missing value for long flag": {
			args:    []string{"--region"},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var flags testFlags
			remaining, positional, err := Sift(&flags, tc.args, tc.passthroughWithArg)

			if tc.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrParse)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantRemaining, remaining, "remaining")
			assert.Equal(t, tc.wantPositional, positional, "positional")
			assert.Equal(t, tc.wantRegion, flags.Region, "region")
			assert.Equal(t, tc.wantProfile, flags.Profile, "profile")
			assert.Equal(t, tc.wantVerbose, flags.Verbose, "verbose")
			assert.Equal(t, tc.wantDebug, flags.Debug, "debug")
		})
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args           []string
		wantPositional []string
		wantRegion     string
		wantProfile    string
		wantVerbose    bool
		wantErr        bool
		errContains    string
	}{
		"valid flags": {
			args:           []string{"--region", "us-west-2", "host"},
			wantRegion:     "us-west-2",
			wantPositional: []string{"host"},
		},
		"all flag types": {
			args:           []string{"-v", "-r", "us-west-2", "--profile", "myprofile", "host"},
			wantVerbose:    true,
			wantRegion:     "us-west-2",
			wantProfile:    "myprofile",
			wantPositional: []string{"host"},
		},
		"unknown short flag rejected": {
			args:        []string{"-x"},
			wantErr:     true,
			errContains: "unknown option -x",
		},
		"unknown long flag rejected": {
			args:        []string{"--unknown"},
			wantErr:     true,
			errContains: "unknown option --unknown",
		},
		"empty args": {
			args:           []string{},
			wantPositional: nil,
		},
		"positional only": {
			args:           []string{"host1", "host2"},
			wantPositional: []string{"host1", "host2"},
		},
		"missing value rejected": {
			args:        []string{"--region"},
			wantErr:     true,
			errContains: "missing value for --region",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var flags testFlags
			positional, err := Parse(&flags, tc.args)

			if tc.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrParse)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantPositional, positional)
			assert.Equal(t, tc.wantRegion, flags.Region)
			assert.Equal(t, tc.wantProfile, flags.Profile)
			assert.Equal(t, tc.wantVerbose, flags.Verbose)
		})
	}
}

func TestSift_EmbeddedStruct(t *testing.T) {
	t.Parallel()

	var flags testEmbedded
	_, _, err := Sift(&flags, []string{"-r", "us-west-2", "-p", "myprofile"}, nil)

	require.NoError(t, err)
	assert.Equal(t, "us-west-2", flags.Region)
	assert.Equal(t, "myprofile", flags.Profile)
}

func TestSift_PanicsOnInvalidTarget(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		target any
	}{
		"nil target":        {target: nil},
		"non-pointer":       {target: testFlags{}},
		"pointer to string": {target: new(string)},
		"pointer to int":    {target: new(int)},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() {
				_, _, _ = Sift(tc.target, []string{}, nil)
			})
		})
	}
}

func TestSift_PanicsOnUnsupportedFieldType(t *testing.T) {
	t.Parallel()

	type badStruct struct {
		Count int `short:"c"`
	}

	assert.Panics(t, func() {
		var flags badStruct
		_, _, _ = Sift(&flags, []string{}, nil)
	})
}

func TestParse_PanicsOnInvalidTarget(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		target any
	}{
		"nil target":  {target: nil},
		"non-pointer": {target: testFlags{}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() {
				_, _ = Parse(tc.target, []string{})
			})
		})
	}
}

func TestSift_LongFlagEqualsEmptyValue(t *testing.T) {
	t.Parallel()

	var flags testFlags
	_, _, err := Sift(&flags, []string{"--region="}, nil)

	require.NoError(t, err)
	assert.Equal(t, "", flags.Region)
}

func TestSift_ComplexPassthrough(t *testing.T) {
	t.Parallel()

	var flags testFlags
	remaining, positional, err := Sift(&flags,
		[]string{"-v", "-o", "opt1", "-L", "8080:localhost:80", "--region", "us-west-2", "host"},
		[]string{"-o", "-L"},
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"-o", "opt1", "-L", "8080:localhost:80"}, remaining)
	assert.Equal(t, []string{"host"}, positional)
	assert.True(t, flags.Verbose)
	assert.Equal(t, "us-west-2", flags.Region)
}

// logLevel implements encoding.TextUnmarshaler for testing custom type support.
type logLevel int

const (
	logLevelInfo logLevel = iota
	logLevelDebug
	logLevelError
)

func (l *logLevel) UnmarshalText(text []byte) error {
	switch string(text) {
	case "", "info":
		*l = logLevelInfo
	case "debug":
		*l = logLevelDebug
	case "error":
		*l = logLevelError
	default:
		return assert.AnError
	}
	return nil
}

// strictLevel is like logLevel but rejects empty strings (matches DstType/AddrType behavior)
type strictLevel int

const (
	strictLevelLow strictLevel = iota
	strictLevelHigh
)

func (s *strictLevel) UnmarshalText(text []byte) error {
	switch string(text) {
	case "low":
		*s = strictLevelLow
	case "high":
		*s = strictLevelHigh
	default:
		return fmt.Errorf("unknown strict level: %q", text)
	}
	return nil
}

func TestSift_TextUnmarshaler(t *testing.T) {
	t.Parallel()

	type customFlags struct {
		Level   logLevel `long:"level"`
		Verbose bool     `short:"v"`
	}

	tests := map[string]struct {
		args      []string
		wantLevel logLevel
		wantErr   bool
	}{
		"valid debug": {
			args:      []string{"--level", "debug"},
			wantLevel: logLevelDebug,
		},
		"valid error": {
			args:      []string{"--level", "error"},
			wantLevel: logLevelError,
		},
		"valid info": {
			args:      []string{"--level", "info"},
			wantLevel: logLevelInfo,
		},
		"empty defaults to info": {
			args:      []string{"--level", ""},
			wantLevel: logLevelInfo,
		},
		"invalid value": {
			args:    []string{"--level", "invalid"},
			wantErr: true,
		},
		"with equals": {
			args:      []string{"--level=debug"},
			wantLevel: logLevelDebug,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var flags customFlags
			_, _, err := Sift(&flags, tc.args, nil)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantLevel, flags.Level)
		})
	}
}

func TestParse_TextUnmarshaler(t *testing.T) {
	t.Parallel()

	type customFlags struct {
		Level logLevel `short:"l" long:"level"`
	}

	tests := map[string]struct {
		args      []string
		wantLevel logLevel
		wantErr   bool
	}{
		"short flag with value": {
			args:      []string{"-l", "debug"},
			wantLevel: logLevelDebug,
		},
		"short flag attached value": {
			args:      []string{"-ldebug"},
			wantLevel: logLevelDebug,
		},
		"invalid short": {
			args:    []string{"-l", "invalid"},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var flags customFlags
			_, err := Parse(&flags, tc.args)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantLevel, flags.Level)
		})
	}
}

func TestSift_PointerToTextUnmarshaler(t *testing.T) {
	t.Parallel()

	type pointerFlags struct {
		Level *logLevel `long:"level"`
		Name  string    `long:"name"`
	}

	tests := map[string]struct {
		args      []string
		wantLevel *logLevel
		wantNil   bool
	}{
		"flag absent - nil": {
			args:    []string{"--name", "test"},
			wantNil: true,
		},
		"flag present - allocated": {
			args:      []string{"--level", "debug"},
			wantLevel: ptrTo(logLevelDebug),
		},
		"flag with equals": {
			args:      []string{"--level=error"},
			wantLevel: ptrTo(logLevelError),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var flags pointerFlags
			_, _, err := Sift(&flags, tc.args, nil)
			require.NoError(t, err)

			if tc.wantNil {
				assert.Nil(t, flags.Level)
			} else {
				require.NotNil(t, flags.Level)
				assert.Equal(t, *tc.wantLevel, *flags.Level)
			}
		})
	}
}

func TestParse_PointerToTextUnmarshaler(t *testing.T) {
	t.Parallel()

	type pointerFlags struct {
		Level *logLevel `short:"l" long:"level"`
	}

	tests := map[string]struct {
		args      []string
		wantLevel *logLevel
		wantNil   bool
		wantErr   bool
	}{
		"short flag with value": {
			args:      []string{"-l", "debug"},
			wantLevel: ptrTo(logLevelDebug),
		},
		"short flag attached": {
			args:      []string{"-lerror"},
			wantLevel: ptrTo(logLevelError),
		},
		"absent is nil": {
			args:    []string{},
			wantNil: true,
		},
		"invalid value": {
			args:    []string{"--level", "invalid"},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var flags pointerFlags
			_, err := Parse(&flags, tc.args)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tc.wantNil {
				assert.Nil(t, flags.Level)
			} else {
				require.NotNil(t, flags.Level)
				assert.Equal(t, *tc.wantLevel, *flags.Level)
			}
		})
	}
}

// ptrTo returns a pointer to the value (generic helper for tests).
func ptrTo[T any](v T) *T {
	return &v
}

func TestSift_PanicsOnUnsupportedPointerType(t *testing.T) {
	t.Parallel()

	// *string does not implement encoding.TextUnmarshaler
	type badStruct struct {
		Value *string `long:"value"`
	}

	assert.Panics(t, func() {
		var flags badStruct
		_, _, _ = Sift(&flags, []string{}, nil)
	})
}

func TestParse_PanicsOnUnsupportedPointerType(t *testing.T) {
	t.Parallel()

	// *int does not implement encoding.TextUnmarshaler
	type badStruct struct {
		Count *int `long:"count"`
	}

	assert.Panics(t, func() {
		var flags badStruct
		_, _ = Parse(&flags, []string{})
	})
}

// TestParse_PointerEmptyStringRejection tests that types rejecting empty strings
// (like DstType and AddrType) correctly propagate errors through pointer fields.
func TestParse_PointerEmptyStringRejection(t *testing.T) {
	t.Parallel()

	type strictFlags struct {
		Level *strictLevel `long:"level"`
	}

	tests := map[string]struct {
		args    []string
		wantErr bool
	}{
		"valid value": {
			args:    []string{"--level", "high"},
			wantErr: false,
		},
		"empty string with equals": {
			args:    []string{"--level="},
			wantErr: true,
		},
		"empty string separate": {
			args:    []string{"--level", ""},
			wantErr: true,
		},
		"absent is nil not error": {
			args:    []string{},
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var flags strictFlags
			_, err := Parse(&flags, tc.args)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unknown strict level")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
