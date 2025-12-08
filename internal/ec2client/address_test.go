package ec2client

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddrType_UnmarshalText(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input   string
		want    AddrType
		wantErr bool
	}{
		"private": {
			input: "private",
			want:  AddrTypePrivate,
		},
		"public": {
			input: "public",
			want:  AddrTypePublic,
		},
		"ipv6": {
			input: "ipv6",
			want:  AddrTypeIPv6,
		},
		"empty string is error": {
			input:   "",
			wantErr: true, // Use *AddrType with nil for auto-detect
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

			var got AddrType
			err := got.UnmarshalText([]byte(tc.input))

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unknown address type")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetInstanceAddr(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		instance types.Instance
		addrType *AddrType // nil = auto-detect
		wantAddr string
		wantType AddrType
		wantErr  bool
	}{
		// nil (auto) mode - prefers public
		"nil prefers public": {
			instance: makeInstance("i-1", withPrivateIP("10.0.0.1"), withPublicIP("1.2.3.4")),
			addrType: nil,
			wantAddr: "1.2.3.4",
			wantType: AddrTypePublic,
		},
		"nil falls back to ipv6": {
			instance: makeInstance("i-1", withIPv6("2001:db8::1")),
			addrType: nil,
			wantAddr: "2001:db8::1",
			wantType: AddrTypeIPv6,
		},
		"nil falls back to private": {
			instance: makeInstance("i-1", withPrivateIP("10.0.0.1")),
			addrType: nil,
			wantAddr: "10.0.0.1",
			wantType: AddrTypePrivate,
		},
		"nil no address": {
			instance: makeInstance("i-1"),
			addrType: nil,
			wantErr:  true,
		},

		// Explicit private
		"explicit private": {
			instance: makeInstance("i-1", withPrivateIP("10.0.0.1"), withPublicIP("1.2.3.4")),
			addrType: addrTypePtr(AddrTypePrivate),
			wantAddr: "10.0.0.1",
			wantType: AddrTypePrivate,
		},
		"explicit private only private": {
			instance: makeInstance("i-1", withPrivateIP("192.168.1.1")),
			addrType: addrTypePtr(AddrTypePrivate),
			wantAddr: "192.168.1.1",
			wantType: AddrTypePrivate,
		},
		"missing private": {
			instance: makeInstance("i-1", withPublicIP("1.2.3.4")),
			addrType: addrTypePtr(AddrTypePrivate),
			wantErr:  true,
		},

		// Explicit public
		"explicit public": {
			instance: makeInstance("i-1", withPrivateIP("10.0.0.1"), withPublicIP("1.2.3.4")),
			addrType: addrTypePtr(AddrTypePublic),
			wantAddr: "1.2.3.4",
			wantType: AddrTypePublic,
		},
		"explicit public only public": {
			instance: makeInstance("i-1", withPublicIP("52.0.0.1")),
			addrType: addrTypePtr(AddrTypePublic),
			wantAddr: "52.0.0.1",
			wantType: AddrTypePublic,
		},
		"missing public": {
			instance: makeInstance("i-1", withPrivateIP("10.0.0.1")),
			addrType: addrTypePtr(AddrTypePublic),
			wantErr:  true,
		},

		// Explicit IPv6
		"explicit ipv6": {
			instance: makeInstance("i-1", withIPv6("2001:db8::1")),
			addrType: addrTypePtr(AddrTypeIPv6),
			wantAddr: "2001:db8::1",
			wantType: AddrTypeIPv6,
		},
		"explicit ipv6 with others": {
			instance: makeInstance("i-1", withPrivateIP("10.0.0.1"), withPublicIP("1.2.3.4"), withIPv6("fe80::1")),
			addrType: addrTypePtr(AddrTypeIPv6),
			wantAddr: "fe80::1",
			wantType: AddrTypeIPv6,
		},
		"missing ipv6": {
			instance: makeInstance("i-1", withPrivateIP("10.0.0.1")),
			addrType: addrTypePtr(AddrTypeIPv6),
			wantErr:  true,
		},

		// All addresses present
		"all addresses nil selects public": {
			instance: makeInstance("i-1", withPrivateIP("10.0.0.1"), withPublicIP("1.2.3.4"), withIPv6("2001:db8::1")),
			addrType: nil,
			wantAddr: "1.2.3.4",
			wantType: AddrTypePublic,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := GetInstanceAddr(tc.instance, tc.addrType)

			if tc.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrNoAddress)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantAddr, result.Addr, "address")
			assert.Equal(t, tc.wantType, result.Type, "type")
		})
	}
}

func TestGetInstanceName(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		instance types.Instance
		want     *string
	}{
		"has name tag": {
			instance: makeInstance("i-1", withNameTag("my-server")),
			want:     aws.String("my-server"),
		},
		"no name tag": {
			instance: makeInstance("i-1"),
			want:     nil,
		},
		"has other tags but no name": {
			instance: makeInstance("i-1", withTag("Environment", "prod"), withTag("Team", "devops")),
			want:     nil,
		},
		"empty name tag": {
			instance: makeInstance("i-1", withNameTag("")),
			want:     aws.String(""),
		},
		"name tag among others": {
			instance: makeInstance("i-1", withTag("Environment", "prod"), withNameTag("web-server"), withTag("Team", "devops")),
			want:     aws.String("web-server"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := GetInstanceName(tc.instance)

			if tc.want == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, *tc.want, *result)
			}
		})
	}
}
