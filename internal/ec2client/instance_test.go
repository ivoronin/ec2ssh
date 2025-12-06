package ec2client

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGuessDestinationType(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		dst  string
		want DstType
	}{
		// Instance ID
		"instance id": {
			dst:  "i-1234567890abcdef0",
			want: DstTypeID,
		},
		"instance id short": {
			dst:  "i-abc123",
			want: DstTypeID,
		},

		// Private DNS names
		"private dns ec2.internal": {
			dst:  "ip-10-0-0-1.ec2.internal",
			want: DstTypePrivateDNSName,
		},
		"private dns compute.internal": {
			dst:  "ip-10-0-0-1.us-west-2.compute.internal",
			want: DstTypePrivateDNSName,
		},
		"private dns short form": {
			dst:  "ip-10-0-0-1",
			want: DstTypePrivateDNSName,
		},

		// Private IPs (RFC 1918)
		"private ip 10.x": {
			dst:  "10.0.0.1",
			want: DstTypePrivateIP,
		},
		"private ip 172.16.x": {
			dst:  "172.16.0.1",
			want: DstTypePrivateIP,
		},
		"private ip 172.31.x": {
			dst:  "172.31.255.255",
			want: DstTypePrivateIP,
		},
		"private ip 192.168.x": {
			dst:  "192.168.1.1",
			want: DstTypePrivateIP,
		},

		// Public IPs
		"public ip 52.x": {
			dst:  "52.10.20.30",
			want: DstTypePublicIP,
		},
		"public ip 8.x": {
			dst:  "8.8.8.8",
			want: DstTypePublicIP,
		},
		"public ip 1.x": {
			dst:  "1.2.3.4",
			want: DstTypePublicIP,
		},

		// IPv6
		"ipv6 full": {
			dst:  "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			want: DstTypeIPv6,
		},
		"ipv6 compressed": {
			dst:  "2001:db8::1",
			want: DstTypeIPv6,
		},
		"ipv6 loopback": {
			dst:  "::1",
			want: DstTypeIPv6,
		},
		"ipv6 link local": {
			dst:  "fe80::1",
			want: DstTypeIPv6,
		},

		// Name tags (fallback)
		"name tag simple": {
			dst:  "my-server",
			want: DstTypeNameTag,
		},
		"name tag with dots": {
			dst:  "web.prod.example",
			want: DstTypeNameTag,
		},
		"name tag with hyphen": {
			dst:  "web-server-01",
			want: DstTypeNameTag,
		},
		"empty string": {
			dst:  "",
			want: DstTypeNameTag,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, GuessDestinationType(tc.dst))
		})
	}
}

func TestClient_GetInstanceByID(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		instanceID  string
		mockSetup   func(*MockEC2API)
		wantID      string
		wantErr     bool
		errContains string
	}{
		"found instance": {
			instanceID: "i-1234567890abcdef0",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					return len(input.InstanceIds) == 1 && input.InstanceIds[0] == "i-1234567890abcdef0"
				})).Return(
					makeDescribeOutput(makeReservation(makeInstance("i-1234567890abcdef0"))),
					nil,
				)
			},
			wantID: "i-1234567890abcdef0",
		},
		"not found - empty result": {
			instanceID: "i-notfound",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(
					makeDescribeOutput(), // empty reservations
					nil,
				)
			},
			wantErr:     true,
			errContains: "no matching instances found",
		},
		"api error": {
			instanceID: "i-error",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("API error"),
				)
			},
			wantErr:     true,
			errContains: "API error",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockEC2 := new(MockEC2API)
			tc.mockSetup(mockEC2)

			client := newTestClient(mockEC2, nil, nil)
			instance, err := client.GetInstanceByID(tc.instanceID)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantID, *instance.InstanceId)
			mockEC2.AssertExpectations(t)
		})
	}
}

func TestClient_GetRunningInstanceByFilter(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		filterName  string
		filterValue string
		mockSetup   func(*MockEC2API)
		wantID      string
		wantErr     bool
	}{
		"found by private ip": {
			filterName:  "private-ip-address",
			filterValue: "10.0.0.1",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					if len(input.Filters) != 2 {
						return false
					}
					hasIPFilter := false
					hasStateFilter := false
					for _, f := range input.Filters {
						if *f.Name == "private-ip-address" && f.Values[0] == "10.0.0.1" {
							hasIPFilter = true
						}
						if *f.Name == "instance-state-name" && f.Values[0] == "running" {
							hasStateFilter = true
						}
					}
					return hasIPFilter && hasStateFilter
				})).Return(
					makeDescribeOutput(makeReservation(makeInstance("i-found", withPrivateIP("10.0.0.1")))),
					nil,
				)
			},
			wantID: "i-found",
		},
		"found by name tag": {
			filterName:  "tag:Name",
			filterValue: "my-server",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(
					makeDescribeOutput(makeReservation(makeInstance("i-named", withNameTag("my-server")))),
					nil,
				)
			},
			wantID: "i-named",
		},
		"no matches": {
			filterName:  "private-ip-address",
			filterValue: "10.0.0.99",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(
					makeDescribeOutput(),
					nil,
				)
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockEC2 := new(MockEC2API)
			tc.mockSetup(mockEC2)

			client := newTestClient(mockEC2, nil, nil)
			instance, err := client.GetRunningInstanceByFilter(tc.filterName, tc.filterValue)

			if tc.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrNoMatches)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantID, *instance.InstanceId)
			mockEC2.AssertExpectations(t)
		})
	}
}

func TestClient_GetInstance(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		dstType     DstType
		destination string
		mockSetup   func(*MockEC2API)
		wantID      string
		wantErr     bool
	}{
		"auto detect instance id": {
			dstType:     DstTypeAuto,
			destination: "i-auto123",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					return len(input.InstanceIds) == 1 && input.InstanceIds[0] == "i-auto123"
				})).Return(
					makeDescribeOutput(makeReservation(makeInstance("i-auto123"))),
					nil,
				)
			},
			wantID: "i-auto123",
		},
		"auto detect private ip": {
			dstType:     DstTypeAuto,
			destination: "10.0.0.5",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					for _, f := range input.Filters {
						if *f.Name == "private-ip-address" && f.Values[0] == "10.0.0.5" {
							return true
						}
					}
					return false
				})).Return(
					makeDescribeOutput(makeReservation(makeInstance("i-byip", withPrivateIP("10.0.0.5")))),
					nil,
				)
			},
			wantID: "i-byip",
		},
		"auto detect public ip": {
			dstType:     DstTypeAuto,
			destination: "54.123.45.67",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					for _, f := range input.Filters {
						if *f.Name == "ip-address" && f.Values[0] == "54.123.45.67" {
							return true
						}
					}
					return false
				})).Return(
					makeDescribeOutput(makeReservation(makeInstance("i-bypubip", withPublicIP("54.123.45.67")))),
					nil,
				)
			},
			wantID: "i-bypubip",
		},
		"auto detect private dns": {
			dstType:     DstTypeAuto,
			destination: "ip-10-0-0-1.ec2.internal",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					for _, f := range input.Filters {
						if *f.Name == "private-dns-name" {
							return true
						}
					}
					return false
				})).Return(
					makeDescribeOutput(makeReservation(makeInstance("i-bydns"))),
					nil,
				)
			},
			wantID: "i-bydns",
		},
		"explicit private dns with wildcard": {
			dstType:     DstTypePrivateDNSName,
			destination: "ip-10-0-0-1",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					for _, f := range input.Filters {
						if *f.Name == "private-dns-name" && f.Values[0] == "ip-10-0-0-1.*" {
							return true
						}
					}
					return false
				})).Return(
					makeDescribeOutput(makeReservation(makeInstance("i-dns"))),
					nil,
				)
			},
			wantID: "i-dns",
		},
		"explicit name tag": {
			dstType:     DstTypeNameTag,
			destination: "my-server",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					for _, f := range input.Filters {
						if *f.Name == "tag:Name" && f.Values[0] == "my-server" {
							return true
						}
					}
					return false
				})).Return(
					makeDescribeOutput(makeReservation(makeInstance("i-named"))),
					nil,
				)
			},
			wantID: "i-named",
		},
		"explicit ipv6": {
			dstType:     DstTypeIPv6,
			destination: "2001:db8::1",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					for _, f := range input.Filters {
						if *f.Name == "ipv6-address" && f.Values[0] == "2001:db8::1" {
							return true
						}
					}
					return false
				})).Return(
					makeDescribeOutput(makeReservation(makeInstance("i-ipv6"))),
					nil,
				)
			},
			wantID: "i-ipv6",
		},
		"not found": {
			dstType:     DstTypeNameTag,
			destination: "nonexistent",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(
					makeDescribeOutput(),
					nil,
				)
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockEC2 := new(MockEC2API)
			tc.mockSetup(mockEC2)

			client := newTestClient(mockEC2, nil, nil)
			instance, err := client.GetInstance(tc.dstType, tc.destination)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantID, *instance.InstanceId)
			mockEC2.AssertExpectations(t)
		})
	}
}

func TestClient_ListInstances(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		mockSetup func(*MockEC2API)
		wantCount int
		wantErr   bool
	}{
		"multiple reservations": {
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(
					makeDescribeOutput(
						makeReservation(makeInstance("i-1"), makeInstance("i-2")),
						makeReservation(makeInstance("i-3")),
					),
					nil,
				)
			},
			wantCount: 3,
		},
		"single reservation multiple instances": {
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(
					makeDescribeOutput(
						makeReservation(makeInstance("i-1"), makeInstance("i-2"), makeInstance("i-3"), makeInstance("i-4")),
					),
					nil,
				)
			},
			wantCount: 4,
		},
		"empty result": {
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(
					makeDescribeOutput(),
					nil,
				)
			},
			wantCount: 0,
		},
		"api error": {
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("API error"),
				)
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockEC2 := new(MockEC2API)
			tc.mockSetup(mockEC2)

			client := newTestClient(mockEC2, nil, nil)
			instances, err := client.ListInstances()

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, instances, tc.wantCount)
			mockEC2.AssertExpectations(t)
		})
	}
}
