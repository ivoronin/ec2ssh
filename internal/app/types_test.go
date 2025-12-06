package app

import (
	"testing"

	"github.com/ivoronin/ec2ssh/internal/ec2client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDstType(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input   string
		want    ec2client.DstType
		wantErr bool
	}{
		"empty string - auto": {
			input: "",
			want:  ec2client.DstTypeAuto,
		},
		"id": {
			input: "id",
			want:  ec2client.DstTypeID,
		},
		"private_ip": {
			input: "private_ip",
			want:  ec2client.DstTypePrivateIP,
		},
		"public_ip": {
			input: "public_ip",
			want:  ec2client.DstTypePublicIP,
		},
		"ipv6": {
			input: "ipv6",
			want:  ec2client.DstTypeIPv6,
		},
		"private_dns": {
			input: "private_dns",
			want:  ec2client.DstTypePrivateDNSName,
		},
		"name_tag": {
			input: "name_tag",
			want:  ec2client.DstTypeNameTag,
		},
		"invalid type": {
			input:   "invalid",
			wantErr: true,
		},
		"unknown type": {
			input:   "unknown_type",
			wantErr: true,
		},
		"partial match": {
			input:   "private", // Not "private_ip" or "private_dns"
			wantErr: true,
		},
		"uppercase rejected": {
			input:   "ID",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseDstType(tc.input)

			if tc.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrUsage)
				assert.Contains(t, err.Error(), "unknown destination type")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseAddrType(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input   string
		want    ec2client.AddrType
		wantErr bool
	}{
		"empty string - auto": {
			input: "",
			want:  ec2client.AddrTypeAuto,
		},
		"private": {
			input: "private",
			want:  ec2client.AddrTypePrivate,
		},
		"public": {
			input: "public",
			want:  ec2client.AddrTypePublic,
		},
		"ipv6": {
			input: "ipv6",
			want:  ec2client.AddrTypeIPv6,
		},
		"invalid type": {
			input:   "invalid",
			wantErr: true,
		},
		"unknown type": {
			input:   "unknown_type",
			wantErr: true,
		},
		"partial match": {
			input:   "priv", // Not "private"
			wantErr: true,
		},
		"uppercase rejected": {
			input:   "PUBLIC",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseAddrType(tc.input)

			if tc.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrUsage)
				assert.Contains(t, err.Error(), "unknown address type")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
