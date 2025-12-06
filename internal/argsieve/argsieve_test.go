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
			remaining, positional, err := Sift(&opts, tt.args, tt.passthrough)

			if tt.wantErr {
				require.ErrorIs(t, err, ErrParse)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOpts, opts)
			assert.Equal(t, tt.wantRemain, remaining)
			assert.Equal(t, tt.wantPos, positional)
		})
	}
}

func TestSift_PanicOnInvalidTarget(t *testing.T) {
	t.Parallel()

	t.Run("non-pointer", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			var opts testOptions
			Sift(opts, nil, nil) // passing value, not pointer
		})
	})

	t.Run("pointer to non-struct", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			var s string
			Sift(&s, nil, nil)
		})
	})

	t.Run("unsupported field type", func(t *testing.T) {
		t.Parallel()

		type badOptions struct {
			Count int `long:"count"`
		}

		assert.Panics(t, func() {
			var opts badOptions
			Sift(&opts, nil, nil)
		})
	})
}

func TestSift_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("lone dash is positional", func(t *testing.T) {
		t.Parallel()

		var opts testOptions
		remaining, positional, err := Sift(&opts, []string{"--debug", "-", "file.txt"}, nil)

		require.NoError(t, err)
		assert.True(t, opts.Debug)
		assert.Empty(t, remaining)
		assert.Equal(t, []string{"-", "file.txt"}, positional)
	})

	t.Run("passthrough at end without value", func(t *testing.T) {
		t.Parallel()

		var opts testOptions
		remaining, positional, err := Sift(&opts, []string{"-o"}, []string{"-o"})

		require.NoError(t, err)
		assert.Equal(t, []string{"-o"}, remaining)
		assert.Empty(t, positional)
	})
}

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		wantOpts testOptions
		wantPos  []string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "known flags work normally",
			args:     []string{"--region", "us-west-2", "--debug"},
			wantOpts: testOptions{Region: "us-west-2", Debug: true},
		},
		{
			name:    "unknown long flag causes error",
			args:    []string{"--unknown", "--debug"},
			wantErr: true,
			errMsg:  "unknown option --unknown",
		},
		{
			name:    "unknown short flag causes error",
			args:    []string{"-N", "--debug"},
			wantErr: true,
			errMsg:  "unknown option -N",
		},
		{
			name:    "unknown in combined short flags causes error",
			args:    []string{"-dXv"},
			wantErr: true,
			errMsg:  "unknown option -X",
		},
		{
			name:     "positional args still work",
			args:     []string{"--debug", "hostname", "cmd"},
			wantOpts: testOptions{Debug: true},
			wantPos:  []string{"hostname", "cmd"},
		},
		{
			name:     "separator treats rest as positional",
			args:     []string{"--debug", "--", "--looks-like-flag"},
			wantOpts: testOptions{Debug: true},
			wantPos:  []string{"--looks-like-flag"},
		},
		{
			name:    "missing value still errors",
			args:    []string{"--region"},
			wantErr: true,
			errMsg:  "missing value for --region",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var opts testOptions
			positional, err := Parse(&opts, tt.args)

			if tt.wantErr {
				require.ErrorIs(t, err, ErrParse)
				assert.Contains(t, err.Error(), tt.errMsg)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOpts, opts)
			assert.Equal(t, tt.wantPos, positional)
		})
	}
}

func TestParse_PanicOnInvalidTarget(t *testing.T) {
	t.Parallel()

	t.Run("non-pointer", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			var opts testOptions
			Parse(opts, nil)
		})
	})

	t.Run("pointer to non-struct", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			var s string
			Parse(&s, nil)
		})
	})
}
