package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewListOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr string
		check   func(t *testing.T, opts *ListOptions)
	}{
		{
			name: "region flag",
			args: []string{"--region", "us-west-2"},
			check: func(t *testing.T, opts *ListOptions) {
				assert.Equal(t, "us-west-2", opts.Region)
			},
		},
		{
			name: "columns flag",
			args: []string{"--list-columns", "ID,NAME,STATE"},
			check: func(t *testing.T, opts *ListOptions) {
				assert.Equal(t, "ID,NAME,STATE", opts.Columns)
			},
		},
		{
			name: "all options",
			args: []string{"--region", "eu-west-1", "--profile", "myprofile", "--list-columns", "ID,TYPE", "--debug"},
			check: func(t *testing.T, opts *ListOptions) {
				assert.Equal(t, "eu-west-1", opts.Region)
				assert.Equal(t, "myprofile", opts.Profile)
				assert.Equal(t, "ID,TYPE", opts.Columns)
				assert.True(t, opts.Debug)
			},
		},
		{
			name:    "unknown option",
			args:    []string{"--use-eice"},
			wantErr: "unknown option",
		},
		{
			name:    "unexpected positional",
			args:    []string{"host"},
			wantErr: "unexpected argument",
		},
		{
			name:    "missing value for --region",
			args:    []string{"--region"},
			wantErr: "missing value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts, err := NewListOptions(tt.args)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, opts)
			}
		})
	}
}

func TestParseListColumns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{"default columns", "", []string{"ID", "NAME", "STATE", "PRIVATE-IP", "PUBLIC-IP"}, false},
		{"custom columns", "ID,NAME", []string{"ID", "NAME"}, false},
		{"lowercase normalized", "id,name", []string{"ID", "NAME"}, false},
		{"with spaces", "ID, NAME, STATE", []string{"ID", "NAME", "STATE"}, false},
		{"invalid column", "ID,INVALID", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseListColumns(tt.input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
