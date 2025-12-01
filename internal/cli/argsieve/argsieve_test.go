package argsieve

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testOptions is a struct for testing with argsieve short/long tags.
type testOptions struct {
	Region  string `short:"r" long:"region"`
	Profile string `long:"profile"`
	Debug   bool   `short:"d" long:"debug"`
	Verbose bool   `short:"v"`
	Login   string `short:"l"`
	Port    string `short:"p" long:"port"`
	NoValue string // no tag - should be ignored
}

func TestSift(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		passthrough []string
		wantOpts    testOptions
		wantRemain  []string
		wantPos     []string
		wantErr     bool
	}{
		// Long options
		{
			name:     "long option with space",
			args:     []string{"--region", "us-west-2"},
			wantOpts: testOptions{Region: "us-west-2"},
		},
		{
			name:     "long option with equals",
			args:     []string{"--region=us-east-1"},
			wantOpts: testOptions{Region: "us-east-1"},
		},
		{
			name:     "bool long flag",
			args:     []string{"--debug"},
			wantOpts: testOptions{Debug: true},
		},
		{
			name:    "missing value for long option",
			args:    []string{"--region"},
			wantErr: true,
		},

		// Short options
		{
			name:     "short option with space",
			args:     []string{"-l", "root"},
			wantOpts: testOptions{Login: "root"},
		},
		{
			name:     "short option attached",
			args:     []string{"-lroot"},
			wantOpts: testOptions{Login: "root"},
		},
		{
			name:     "combined short bool flags",
			args:     []string{"-dv"},
			wantOpts: testOptions{Debug: true, Verbose: true},
		},
		{
			name:     "combined short flags with value",
			args:     []string{"-dvlroot"},
			wantOpts: testOptions{Debug: true, Verbose: true, Login: "root"},
		},
		{
			name:    "missing value for short option",
			args:    []string{"-l"},
			wantErr: true,
		},

		// Unknown flags
		{
			name:       "unknown short flag",
			args:       []string{"-N", "--debug"},
			wantOpts:   testOptions{Debug: true},
			wantRemain: []string{"-N"},
		},
		{
			name:       "unknown long flag",
			args:       []string{"--unknown", "--debug"},
			wantOpts:   testOptions{Debug: true},
			wantRemain: []string{"--unknown"},
		},

		// Passthrough flags
		{
			name:        "passthrough long with arg",
			args:        []string{"--url", "http://example.com", "--debug"},
			passthrough: []string{"--url"},
			wantOpts:    testOptions{Debug: true},
			wantRemain:  []string{"--url", "http://example.com"},
		},
		{
			name:        "passthrough long with equals",
			args:        []string{"--url=http://example.com", "--debug"},
			passthrough: []string{"--url"},
			wantOpts:    testOptions{Debug: true},
			wantRemain:  []string{"--url=http://example.com"},
		},
		{
			name:        "passthrough short with arg",
			args:        []string{"-o", "output.txt", "--debug"},
			passthrough: []string{"-o"},
			wantOpts:    testOptions{Debug: true},
			wantRemain:  []string{"-o", "output.txt"},
		},
		{
			name:        "passthrough short attached",
			args:        []string{"-ooutput.txt", "--debug"},
			passthrough: []string{"-o"},
			wantOpts:    testOptions{Debug: true},
			wantRemain:  []string{"-ooutput.txt"},
		},
		{
			name:        "passthrough short consumes next arg even if flag-like",
			args:        []string{"-o", "--debug", "hostname"},
			passthrough: []string{"-o"},
			wantOpts:    testOptions{Debug: false}, // --debug is consumed by -o, not parsed
			wantRemain:  []string{"-o", "--debug"},
			wantPos:     []string{"hostname"},
		},

		// Positional arguments
		{
			name:     "positional args",
			args:     []string{"--debug", "hostname", "cmd", "arg1"},
			wantOpts: testOptions{Debug: true},
			wantPos:  []string{"hostname", "cmd", "arg1"},
		},
		{
			name:    "only positional",
			args:    []string{"hostname", "cmd"},
			wantPos: []string{"hostname", "cmd"},
		},

		// Separator --
		{
			name:     "separator treats rest as positional",
			args:     []string{"--debug", "--", "--not-a-flag", "positional"},
			wantOpts: testOptions{Debug: true},
			wantPos:  []string{"--not-a-flag", "positional"},
		},

		// Edge cases
		{
			name: "empty args",
			args: []string{},
		},
		{
			name:     "duplicate option (last wins)",
			args:     []string{"--region", "first", "--region", "second"},
			wantOpts: testOptions{Region: "second"},
		},
		{
			name:        "without passthrough - value becomes positional",
			args:        []string{"--url", "http://example.com"},
			wantRemain:  []string{"--url"},
			wantPos:     []string{"http://example.com"},
		},
		{
			name:        "with passthrough - value stays with flag",
			args:        []string{"--url", "http://example.com"},
			passthrough: []string{"--url"},
			wantRemain:  []string{"--url", "http://example.com"},
		},

		// Complex mix
		{
			name: "complex mix of all types",
			args: []string{
				"--region", "us-west-2",
				"-dv",
				"-lroot",
				"-N",
				"--url", "http://example.com",
				"hostname",
				"cmd",
			},
			passthrough: []string{"--url"},
			wantOpts:    testOptions{Region: "us-west-2", Debug: true, Verbose: true, Login: "root"},
			wantRemain:  []string{"-N", "--url", "http://example.com"},
			wantPos:     []string{"hostname", "cmd"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var opts testOptions
			sieve := New(&opts, tt.passthrough)
			remaining, positional, err := sieve.Sift(tt.args)

			if tt.wantErr {
				require.ErrorIs(t, err, ErrSift)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOpts, opts)
			assert.Equal(t, tt.wantRemain, remaining)
			assert.Equal(t, tt.wantPos, positional)
		})
	}
}

func TestNew_PanicOnInvalidTarget(t *testing.T) {
	t.Parallel()

	t.Run("non-pointer", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			var opts testOptions
			New(opts, nil) // passing value, not pointer
		})
	})

	t.Run("pointer to non-struct", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			var s string
			New(&s, nil)
		})
	})

	t.Run("unsupported field type", func(t *testing.T) {
		t.Parallel()

		type badOptions struct {
			Count int `long:"count"`
		}

		assert.Panics(t, func() {
			var opts badOptions
			New(&opts, nil)
		})
	})
}

func TestSift_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("lone dash is positional", func(t *testing.T) {
		t.Parallel()

		var opts testOptions
		sieve := New(&opts, nil)
		remaining, positional, err := sieve.Sift([]string{"--debug", "-", "file.txt"})

		require.NoError(t, err)
		assert.True(t, opts.Debug)
		assert.Empty(t, remaining)
		assert.Equal(t, []string{"-", "file.txt"}, positional)
	})

	t.Run("passthrough at end without value", func(t *testing.T) {
		t.Parallel()

		var opts testOptions
		sieve := New(&opts, []string{"-o"})
		remaining, positional, err := sieve.Sift([]string{"-o"})

		require.NoError(t, err)
		assert.Equal(t, []string{"-o"}, remaining)
		assert.Empty(t, positional)
	})
}
