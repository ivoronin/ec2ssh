package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewListOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args        []string
		wantRegion  string
		wantProfile string
		wantColumns string
		wantDebug   bool
		wantErr     bool
		errContains string
	}{
		// Valid cases
		"no args": {
			args: []string{},
		},
		"with region": {
			args:       []string{"--region", "us-west-2"},
			wantRegion: "us-west-2",
		},
		"with profile": {
			args:        []string{"--profile", "myprofile"},
			wantProfile: "myprofile",
		},
		"with columns": {
			args:        []string{"--list-columns", "ID,NAME"},
			wantColumns: "ID,NAME",
		},
		"with debug": {
			args:      []string{"--debug"},
			wantDebug: true,
		},
		"all options": {
			args:        []string{"--region", "eu-west-1", "--profile", "prod", "--list-columns", "ID,NAME,STATE", "--debug"},
			wantRegion:  "eu-west-1",
			wantProfile: "prod",
			wantColumns: "ID,NAME,STATE",
			wantDebug:   true,
		},

		// Error cases
		"unexpected positional": {
			args:        []string{"extra"},
			wantErr:     true,
			errContains: "unexpected argument",
		},
		"unknown flag": {
			args:        []string{"--unknown"},
			wantErr:     true,
			errContains: "unknown option",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			options, err := NewListOptions(tc.args)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, options)

			assert.Equal(t, tc.wantRegion, options.Region)
			assert.Equal(t, tc.wantProfile, options.Profile)
			assert.Equal(t, tc.wantColumns, options.Columns)
			assert.Equal(t, tc.wantDebug, options.Debug)
		})
	}
}

func TestParseListColumns(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input   string
		want    []string
		wantErr bool
	}{
		"default - empty string": {
			input: "",
			want:  []string{"ID", "NAME", "STATE", "PRIVATE-IP", "PUBLIC-IP"},
		},
		"single column": {
			input: "ID",
			want:  []string{"ID"},
		},
		"multiple columns": {
			input: "ID,NAME,STATE",
			want:  []string{"ID", "NAME", "STATE"},
		},
		"all columns": {
			input: "ID,NAME,STATE,TYPE,AZ,PRIVATE-IP,PUBLIC-IP,IPV6,PRIVATE-DNS,PUBLIC-DNS",
			want:  []string{"ID", "NAME", "STATE", "TYPE", "AZ", "PRIVATE-IP", "PUBLIC-IP", "IPV6", "PRIVATE-DNS", "PUBLIC-DNS"},
		},
		"case insensitive - lowercase": {
			input: "id,name,state",
			want:  []string{"ID", "NAME", "STATE"},
		},
		"case insensitive - mixed": {
			input: "Id,NaMe,STATE",
			want:  []string{"ID", "NAME", "STATE"},
		},
		"with spaces": {
			input: "ID, NAME, STATE",
			want:  []string{"ID", "NAME", "STATE"},
		},
		"invalid column": {
			input:   "ID,INVALID",
			wantErr: true,
		},
		"unknown column": {
			input:   "UNKNOWN",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := parseListColumns(tc.input)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestWriteInstanceList(t *testing.T) {
	t.Parallel()

	// Helper to create test instances
	makeTestInstance := func(id, name, state, privateIP, publicIP string) types.Instance {
		inst := types.Instance{
			InstanceId:       aws.String(id),
			PrivateIpAddress: awsPtrOrNil(privateIP),
			PublicIpAddress:  awsPtrOrNil(publicIP),
		}
		if state != "" {
			inst.State = &types.InstanceState{Name: types.InstanceStateName(state)}
		}
		if name != "" {
			inst.Tags = []types.Tag{{Key: aws.String("Name"), Value: aws.String(name)}}
		}
		return inst
	}

	tests := map[string]struct {
		instances []types.Instance
		columns   []string
		wantLines []string
	}{
		"empty instances": {
			instances: []types.Instance{},
			columns:   []string{"ID", "NAME"},
			wantLines: []string{"ID", "NAME"},
		},
		"single instance all values": {
			instances: []types.Instance{
				makeTestInstance("i-123", "my-server", "running", "10.0.0.1", "52.0.0.1"),
			},
			columns:   []string{"ID", "NAME", "STATE", "PRIVATE-IP", "PUBLIC-IP"},
			wantLines: []string{"ID", "i-123", "my-server", "running", "10.0.0.1", "52.0.0.1"},
		},
		"missing values show dash": {
			instances: []types.Instance{
				makeTestInstance("i-123", "", "", "10.0.0.1", ""),
			},
			columns:   []string{"ID", "NAME", "STATE", "PRIVATE-IP", "PUBLIC-IP"},
			wantLines: []string{"ID", "i-123", "-", "-", "10.0.0.1", "-"},
		},
		"multiple instances": {
			instances: []types.Instance{
				makeTestInstance("i-123", "server-1", "running", "10.0.0.1", ""),
				makeTestInstance("i-456", "server-2", "stopped", "10.0.0.2", "52.0.0.2"),
			},
			columns:   []string{"ID", "NAME", "STATE"},
			wantLines: []string{"ID", "i-123", "server-1", "running", "i-456", "server-2", "stopped"},
		},
		"custom columns": {
			instances: []types.Instance{
				makeTestInstance("i-123", "my-server", "running", "10.0.0.1", ""),
			},
			columns:   []string{"NAME", "PRIVATE-IP"},
			wantLines: []string{"NAME", "PRIVATE-IP", "my-server", "10.0.0.1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := writeInstanceList(&buf, tc.instances, tc.columns)
			require.NoError(t, err)

			output := buf.String()
			for _, wantLine := range tc.wantLines {
				assert.Contains(t, output, wantLine)
			}
		})
	}
}

func TestWriteInstanceList_AllColumns(t *testing.T) {
	t.Parallel()

	inst := types.Instance{
		InstanceId:       aws.String("i-123"),
		InstanceType:     types.InstanceTypeT2Micro,
		PrivateIpAddress: aws.String("10.0.0.1"),
		PublicIpAddress:  aws.String("52.0.0.1"),
		Ipv6Address:      aws.String("2001:db8::1"),
		PrivateDnsName:   aws.String("ip-10-0-0-1.ec2.internal"),
		PublicDnsName:    aws.String("ec2-52-0-0-1.compute-1.amazonaws.com"),
		State:            &types.InstanceState{Name: types.InstanceStateNameRunning},
		Placement:        &types.Placement{AvailabilityZone: aws.String("us-east-1a")},
		Tags:             []types.Tag{{Key: aws.String("Name"), Value: aws.String("my-server")}},
	}

	columns := []string{"ID", "NAME", "STATE", "TYPE", "AZ", "PRIVATE-IP", "PUBLIC-IP", "IPV6", "PRIVATE-DNS", "PUBLIC-DNS"}

	var buf bytes.Buffer
	err := writeInstanceList(&buf, []types.Instance{inst}, columns)
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have header + 1 data row
	assert.Len(t, lines, 2)

	// Check all values present
	assert.Contains(t, output, "i-123")
	assert.Contains(t, output, "my-server")
	assert.Contains(t, output, "running")
	assert.Contains(t, output, "t2.micro")
	assert.Contains(t, output, "us-east-1a")
	assert.Contains(t, output, "10.0.0.1")
	assert.Contains(t, output, "52.0.0.1")
	assert.Contains(t, output, "2001:db8::1")
	assert.Contains(t, output, "ip-10-0-0-1.ec2.internal")
	assert.Contains(t, output, "ec2-52-0-0-1.compute-1.amazonaws.com")
}

// Helper to create pointer only if string is non-empty
func awsPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return aws.String(s)
}
