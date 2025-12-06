package ec2client

import (
	"errors"
	"strings"
	"testing"

	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetEICEByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		eiceID     string
		setupMock  func(*MockEC2API)
		wantID     string
		wantErr    bool
		errContain string
	}{
		{
			name:   "success - endpoint found",
			eiceID: "eice-0123456789abcdef0",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.MatchedBy(func(input *awsec2.DescribeInstanceConnectEndpointsInput) bool {
					return len(input.InstanceConnectEndpointIds) == 1 &&
						input.InstanceConnectEndpointIds[0] == "eice-0123456789abcdef0"
				})).Return(describeEICEOutput(
					makeEICE("eice-0123456789abcdef0", "eice.us-east-1.amazonaws.com", "vpc-123", "subnet-123"),
				), nil)
			},
			wantID: "eice-0123456789abcdef0",
		},
		{
			name:   "error - endpoint not found",
			eiceID: "eice-nonexistent",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
					describeEICEOutput(), nil)
			},
			wantErr:    true,
			errContain: "no matching instances found",
		},
		{
			name:   "error - API error",
			eiceID: "eice-test",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
					nil, errors.New("AccessDenied"))
			},
			wantErr:    true,
			errContain: "AccessDenied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockEC2 := new(MockEC2API)
			tt.setupMock(mockEC2)
			client := newTestClient(mockEC2, nil, nil)

			eice, err := client.getEICEByID(tt.eiceID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				mockEC2.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantID, *eice.InstanceConnectEndpointId)
			mockEC2.AssertExpectations(t)
		})
	}
}

func TestGuessEICEByVPCAndSubnet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		vpcID      string
		subnetID   string
		setupMock  func(*MockEC2API)
		wantID     string
		wantErr    bool
		errContain string
	}{
		{
			name:     "success - exact subnet match",
			vpcID:    "vpc-123",
			subnetID: "subnet-456",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.MatchedBy(func(input *awsec2.DescribeInstanceConnectEndpointsInput) bool {
					// Verify VPC filter is applied
					for _, f := range input.Filters {
						if *f.Name == "vpc-id" && f.Values[0] == "vpc-123" {
							return true
						}
					}
					return false
				})).Return(describeEICEOutput(
					makeEICE("eice-different", "dns1.com", "vpc-123", "subnet-other"),
					makeEICE("eice-exact", "dns2.com", "vpc-123", "subnet-456"),
				), nil)
			},
			wantID: "eice-exact",
		},
		{
			name:     "success - falls back to VPC match when no subnet match",
			vpcID:    "vpc-123",
			subnetID: "subnet-nomatch",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(describeEICEOutput(
					makeEICE("eice-vpc", "dns.com", "vpc-123", "subnet-other"),
				), nil)
			},
			wantID: "eice-vpc",
		},
		{
			name:     "error - no endpoints in VPC",
			vpcID:    "vpc-empty",
			subnetID: "subnet-any",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
					describeEICEOutput(), nil)
			},
			wantErr:    true,
			errContain: "no matching instances found",
		},
		{
			name:     "error - API error",
			vpcID:    "vpc-123",
			subnetID: "subnet-456",
			setupMock: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
					nil, errors.New("rate limited"))
			},
			wantErr:    true,
			errContain: "rate limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockEC2 := new(MockEC2API)
			tt.setupMock(mockEC2)
			client := newTestClient(mockEC2, nil, nil)

			eice, err := client.GuessEICEByVPCAndSubnet(tt.vpcID, tt.subnetID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				mockEC2.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantID, *eice.InstanceConnectEndpointId)
			mockEC2.AssertExpectations(t)
		})
	}
}

func TestCreateEICETunnelURI_Success(t *testing.T) {
	t.Parallel()

	mockEC2 := new(MockEC2API)
	mockEC2.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.MatchedBy(func(input *awsec2.DescribeInstanceConnectEndpointsInput) bool {
		return len(input.InstanceConnectEndpointIds) == 1 &&
			input.InstanceConnectEndpointIds[0] == "eice-explicit"
	})).Return(describeEICEOutput(
		makeEICE("eice-explicit", "eice.example.com", "vpc-123", "subnet-456"),
	), nil)

	client := newTestClient(mockEC2, nil, nil)

	uri, err := client.CreateEICETunnelURI("10.0.0.1", "22", "eice-explicit")

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(uri, "wss://eice.example.com/openTunnel"))
	assert.Contains(t, uri, "instanceConnectEndpointId=eice-explicit")
	assert.Contains(t, uri, "privateIpAddress=10.0.0.1")
	assert.Contains(t, uri, "remotePort=22")
	mockEC2.AssertExpectations(t)
}

func TestCreateEICETunnelURI_EICELookupFails(t *testing.T) {
	t.Parallel()

	mockEC2 := new(MockEC2API)
	mockEC2.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
		nil, errors.New("AccessDenied"))

	client := newTestClient(mockEC2, nil, nil)

	_, err := client.CreateEICETunnelURI("10.0.0.1", "22", "eice-test")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "AccessDenied")
	mockEC2.AssertExpectations(t)
}

func TestCreateEICETunnelURI_NoEICEFound(t *testing.T) {
	t.Parallel()

	mockEC2 := new(MockEC2API)
	mockEC2.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
		describeEICEOutput(), nil)

	client := newTestClient(mockEC2, nil, nil)

	_, err := client.CreateEICETunnelURI("10.0.0.1", "22", "eice-nonexistent")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no matching instances found")
	mockEC2.AssertExpectations(t)
}
