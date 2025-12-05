package ec2client

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInstanceAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		instance types.Instance
		addrType AddrType
		wantAddr string
		wantErr  bool
	}{
		// Auto type tests - priority: private > public > IPv6
		{
			name:     "auto - prefers private when available",
			instance: makeInstance("i-test", withPrivateIP("10.0.0.1"), withPublicIP("54.1.2.3")),
			addrType: AddrTypeAuto,
			wantAddr: "10.0.0.1",
		},
		{
			name:     "auto - falls back to public when no private",
			instance: makeInstance("i-test", withPublicIP("54.1.2.3")),
			addrType: AddrTypeAuto,
			wantAddr: "54.1.2.3",
		},
		{
			name:     "auto - falls back to IPv6 when no IPv4",
			instance: makeInstance("i-test", withIPv6("2001:db8::1")),
			addrType: AddrTypeAuto,
			wantAddr: "2001:db8::1",
		},
		{
			name:     "auto - error when no addresses",
			instance: makeInstance("i-test"),
			addrType: AddrTypeAuto,
			wantErr:  true,
		},
		{
			name:     "auto - private takes priority over IPv6",
			instance: makeInstance("i-test", withPrivateIP("10.0.0.1"), withIPv6("2001:db8::1")),
			addrType: AddrTypeAuto,
			wantAddr: "10.0.0.1",
		},
		{
			name:     "auto - public takes priority over IPv6",
			instance: makeInstance("i-test", withPublicIP("54.1.2.3"), withIPv6("2001:db8::1")),
			addrType: AddrTypeAuto,
			wantAddr: "54.1.2.3",
		},

		// Explicit private type tests
		{
			name:     "private - success",
			instance: makeInstance("i-test", withPrivateIP("10.0.0.1")),
			addrType: AddrTypePrivate,
			wantAddr: "10.0.0.1",
		},
		{
			name:     "private - success with all addresses",
			instance: makeInstance("i-test", withPrivateIP("10.0.0.1"), withPublicIP("54.1.2.3"), withIPv6("2001:db8::1")),
			addrType: AddrTypePrivate,
			wantAddr: "10.0.0.1",
		},
		{
			name:     "private - error when missing",
			instance: makeInstance("i-test", withPublicIP("54.1.2.3")),
			addrType: AddrTypePrivate,
			wantErr:  true,
		},

		// Explicit public type tests
		{
			name:     "public - success",
			instance: makeInstance("i-test", withPublicIP("54.1.2.3")),
			addrType: AddrTypePublic,
			wantAddr: "54.1.2.3",
		},
		{
			name:     "public - success with all addresses",
			instance: makeInstance("i-test", withPrivateIP("10.0.0.1"), withPublicIP("54.1.2.3"), withIPv6("2001:db8::1")),
			addrType: AddrTypePublic,
			wantAddr: "54.1.2.3",
		},
		{
			name:     "public - error when missing",
			instance: makeInstance("i-test", withPrivateIP("10.0.0.1")),
			addrType: AddrTypePublic,
			wantErr:  true,
		},

		// Explicit IPv6 type tests
		{
			name:     "IPv6 - success",
			instance: makeInstance("i-test", withIPv6("2001:db8::1")),
			addrType: AddrTypeIPv6,
			wantAddr: "2001:db8::1",
		},
		{
			name:     "IPv6 - success with all addresses",
			instance: makeInstance("i-test", withPrivateIP("10.0.0.1"), withPublicIP("54.1.2.3"), withIPv6("2001:db8::1")),
			addrType: AddrTypeIPv6,
			wantAddr: "2001:db8::1",
		},
		{
			name:     "IPv6 - error when missing",
			instance: makeInstance("i-test", withPrivateIP("10.0.0.1")),
			addrType: AddrTypeIPv6,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			addr, err := GetInstanceAddr(tt.instance, tt.addrType)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrNoAddress)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantAddr, addr)
		})
	}
}

func TestGetInstanceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		instance types.Instance
		want     *string
	}{
		{
			name:     "has Name tag",
			instance: makeInstance("i-test", withNameTag("my-server")),
			want:     aws.String("my-server"),
		},
		{
			name:     "no Name tag",
			instance: makeInstance("i-test"),
			want:     nil,
		},
		{
			name: "has other tags but not Name",
			instance: func() types.Instance {
				i := makeInstance("i-test")
				i.Tags = []types.Tag{{Key: aws.String("Environment"), Value: aws.String("prod")}}
				return i
			}(),
			want: nil,
		},
		{
			name: "multiple tags including Name",
			instance: func() types.Instance {
				i := makeInstance("i-test")
				i.Tags = []types.Tag{
					{Key: aws.String("Environment"), Value: aws.String("prod")},
					{Key: aws.String("Name"), Value: aws.String("my-server")},
					{Key: aws.String("Team"), Value: aws.String("platform")},
				}
				return i
			}(),
			want: aws.String("my-server"),
		},
		{
			name: "Name tag with empty value",
			instance: func() types.Instance {
				i := makeInstance("i-test")
				i.Tags = []types.Tag{{Key: aws.String("Name"), Value: aws.String("")}}
				return i
			}(),
			want: aws.String(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := GetInstanceName(tt.instance)

			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, *tt.want, *got)
			}
		})
	}
}
