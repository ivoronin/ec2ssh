package ec2client

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGuessDestinationType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dst  string
		want DstType
	}{
		// Instance ID detection
		{"instance ID full", "i-0123456789abcdef0", DstTypeID},
		{"instance ID short", "i-abc123", DstTypeID},

		// Private DNS name detection
		{"private DNS ec2.internal", "ip-10-0-0-1.ec2.internal", DstTypePrivateDNSName},
		{"private DNS compute.internal", "ip-10-0-0-1.us-west-2.compute.internal", DstTypePrivateDNSName},
		{"private DNS short form", "ip-10-0-0-1", DstTypePrivateDNSName},

		// IPv4 private detection
		{"private IPv4 10.x", "10.0.0.1", DstTypePrivateIP},
		{"private IPv4 172.16.x", "172.16.0.1", DstTypePrivateIP},
		{"private IPv4 192.168.x", "192.168.1.1", DstTypePrivateIP},

		// IPv4 public detection
		{"public IPv4", "54.123.45.67", DstTypePublicIP},
		{"public IPv4 edge", "8.8.8.8", DstTypePublicIP},

		// IPv6 detection
		{"IPv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", DstTypeIPv6},
		{"IPv6 compressed", "::1", DstTypeIPv6},
		{"IPv6 AWS format", "2600:1f18:abc:def0::1", DstTypeIPv6},

		// Name tag (default fallback)
		{"name tag simple", "web-server", DstTypeNameTag},
		{"name tag with dashes", "my-app-prod-01", DstTypeNameTag},
		{"name tag with numbers", "server123", DstTypeNameTag},
		{"empty string", "", DstTypeNameTag},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := GuessDestinationType(tt.dst)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetInstanceByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		instanceID string
		setupMock  func(*MockEC2API)
		wantID     string
		wantErr    bool
		errContain string
	}{
		{
			name:       "success - instance found",
			instanceID: "i-0123456789abcdef0",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					return len(input.InstanceIds) == 1 && input.InstanceIds[0] == "i-0123456789abcdef0"
				})).Return(describeInstancesOutput(makeInstance("i-0123456789abcdef0")), nil)
			},
			wantID: "i-0123456789abcdef0",
		},
		{
			name:       "error - instance not found",
			instanceID: "i-nonexistent",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(describeInstancesOutput(), nil)
			},
			wantErr:    true,
			errContain: "no matching instances found",
		},
		{
			name:       "error - AWS API error",
			instanceID: "i-0123456789abcdef0",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(nil, errors.New("AWS API error"))
			},
			wantErr:    true,
			errContain: "AWS API error",
		},
		{
			name:       "success - multiple reservations returns first instance",
			instanceID: "i-first",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(&ec2.DescribeInstancesOutput{
					Reservations: []types.Reservation{
						{Instances: []types.Instance{makeInstance("i-first")}},
						{Instances: []types.Instance{makeInstance("i-second")}},
					},
				}, nil)
			},
			wantID: "i-first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockEC2 := new(MockEC2API)
			tt.setupMock(mockEC2)
			client := newTestClient(mockEC2, nil, nil)

			instance, err := client.GetInstanceByID(tt.instanceID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				mockEC2.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantID, *instance.InstanceId)
			mockEC2.AssertExpectations(t)
		})
	}
}

func TestGetRunningInstanceByFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		filterName  string
		filterValue string
		setupMock   func(*MockEC2API)
		wantID      string
		wantErr     bool
		errContain  string
	}{
		{
			name:        "success - find by private IP",
			filterName:  "private-ip-address",
			filterValue: "10.0.0.5",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					if len(input.Filters) != 2 {
						return false
					}
					hasFilter := false
					hasRunning := false
					for _, f := range input.Filters {
						if *f.Name == "private-ip-address" && f.Values[0] == "10.0.0.5" {
							hasFilter = true
						}
						if *f.Name == "instance-state-name" && f.Values[0] == "running" {
							hasRunning = true
						}
					}
					return hasFilter && hasRunning
				})).Return(describeInstancesOutput(makeInstance("i-found", withPrivateIP("10.0.0.5"))), nil)
			},
			wantID: "i-found",
		},
		{
			name:        "success - find by name tag",
			filterName:  "tag:Name",
			filterValue: "web-server",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(
					describeInstancesOutput(makeInstance("i-webserver", withNameTag("web-server"))), nil)
			},
			wantID: "i-webserver",
		},
		{
			name:        "error - no running instances match",
			filterName:  "tag:Name",
			filterValue: "nonexistent",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(describeInstancesOutput(), nil)
			},
			wantErr:    true,
			errContain: "no matching instances found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockEC2 := new(MockEC2API)
			tt.setupMock(mockEC2)
			client := newTestClient(mockEC2, nil, nil)

			instance, err := client.GetRunningInstanceByFilter(tt.filterName, tt.filterValue)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				mockEC2.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantID, *instance.InstanceId)
			mockEC2.AssertExpectations(t)
		})
	}
}

func TestListInstances(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*MockEC2API)
		wantCount int
		wantErr   bool
	}{
		{
			name: "success - multiple instances across reservations",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(&ec2.DescribeInstancesOutput{
					Reservations: []types.Reservation{
						{Instances: []types.Instance{makeInstance("i-1"), makeInstance("i-2")}},
						{Instances: []types.Instance{makeInstance("i-3")}},
					},
				}, nil)
			},
			wantCount: 3,
		},
		{
			name: "success - empty result",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(describeInstancesOutput(), nil)
			},
			wantCount: 0,
		},
		{
			name: "error - AWS API failure",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.Anything).Return(nil, errors.New("rate limit exceeded"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockEC2 := new(MockEC2API)
			tt.setupMock(mockEC2)
			client := newTestClient(mockEC2, nil, nil)

			instances, err := client.ListInstances()

			if tt.wantErr {
				require.Error(t, err)
				mockEC2.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			assert.Len(t, instances, tt.wantCount)
			mockEC2.AssertExpectations(t)
		})
	}
}

func TestGetInstance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		dstType     DstType
		destination string
		setupMock   func(*MockEC2API)
		wantID      string
		wantErr     bool
	}{
		{
			name:        "auto-detect instance ID",
			dstType:     DstTypeAuto,
			destination: "i-0123456789abcdef0",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					return len(input.InstanceIds) == 1 && input.InstanceIds[0] == "i-0123456789abcdef0"
				})).Return(describeInstancesOutput(makeInstance("i-0123456789abcdef0")), nil)
			},
			wantID: "i-0123456789abcdef0",
		},
		{
			name:        "explicit ID type",
			dstType:     DstTypeID,
			destination: "i-explicit",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					return len(input.InstanceIds) == 1
				})).Return(describeInstancesOutput(makeInstance("i-explicit")), nil)
			},
			wantID: "i-explicit",
		},
		{
			name:        "private IP filter",
			dstType:     DstTypePrivateIP,
			destination: "10.0.0.100",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					for _, f := range input.Filters {
						if *f.Name == "private-ip-address" && f.Values[0] == "10.0.0.100" {
							return true
						}
					}
					return false
				})).Return(describeInstancesOutput(makeInstance("i-byip")), nil)
			},
			wantID: "i-byip",
		},
		{
			name:        "public IP filter",
			dstType:     DstTypePublicIP,
			destination: "54.1.2.3",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					for _, f := range input.Filters {
						if *f.Name == "ip-address" && f.Values[0] == "54.1.2.3" {
							return true
						}
					}
					return false
				})).Return(describeInstancesOutput(makeInstance("i-bypubip")), nil)
			},
			wantID: "i-bypubip",
		},
		{
			name:        "private DNS short form adds wildcard",
			dstType:     DstTypePrivateDNSName,
			destination: "ip-10-0-0-1",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					for _, f := range input.Filters {
						if *f.Name == "private-dns-name" && f.Values[0] == "ip-10-0-0-1.*" {
							return true
						}
					}
					return false
				})).Return(describeInstancesOutput(makeInstance("i-bydns")), nil)
			},
			wantID: "i-bydns",
		},
		{
			name:        "private DNS full form no wildcard",
			dstType:     DstTypePrivateDNSName,
			destination: "ip-10-0-0-1.ec2.internal",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					for _, f := range input.Filters {
						if *f.Name == "private-dns-name" && f.Values[0] == "ip-10-0-0-1.ec2.internal" {
							return true
						}
					}
					return false
				})).Return(describeInstancesOutput(makeInstance("i-bydns")), nil)
			},
			wantID: "i-bydns",
		},
		{
			name:        "name tag filter",
			dstType:     DstTypeNameTag,
			destination: "web-server",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					for _, f := range input.Filters {
						if *f.Name == "tag:Name" && f.Values[0] == "web-server" {
							return true
						}
					}
					return false
				})).Return(describeInstancesOutput(makeInstance("i-byname")), nil)
			},
			wantID: "i-byname",
		},
		{
			name:        "IPv6 filter",
			dstType:     DstTypeIPv6,
			destination: "2001:db8::1",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstances", mock.Anything, mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
					for _, f := range input.Filters {
						if *f.Name == "ipv6-address" && f.Values[0] == "2001:db8::1" {
							return true
						}
					}
					return false
				})).Return(describeInstancesOutput(makeInstance("i-byipv6")), nil)
			},
			wantID: "i-byipv6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockEC2 := new(MockEC2API)
			tt.setupMock(mockEC2)
			client := newTestClient(mockEC2, nil, nil)

			instance, err := client.GetInstance(tt.dstType, tt.destination)

			if tt.wantErr {
				require.Error(t, err)
				mockEC2.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantID, *instance.InstanceId)
			mockEC2.AssertExpectations(t)
		})
	}
}

// Helper to assert context passed correctly
func assertContextPassed(t *testing.T, ctx context.Context) bool {
	return ctx != nil
}
