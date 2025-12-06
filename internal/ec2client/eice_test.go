package ec2client

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestClient_getEICEByID(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		eiceID      string
		mockSetup   func(*MockEC2API)
		wantID      string
		wantErr     bool
		errContains string
	}{
		"found endpoint": {
			eiceID: "eice-123456789",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
					makeEICEOutput(makeEICE("eice-123456789", "vpc-123", "subnet-456", "eice.example.com")),
					nil,
				)
			},
			wantID: "eice-123456789",
		},
		"not found - empty result": {
			eiceID: "eice-notfound",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
					makeEICEOutput(),
					nil,
				)
			},
			wantErr:     true,
			errContains: "no matching instances found",
		},
		"api error": {
			eiceID: "eice-error",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("API error"),
				)
			},
			wantErr:     true,
			errContains: "API error",
		},
		"returns first of multiple": {
			eiceID: "eice-first",
			mockSetup: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
					makeEICEOutput(
						makeEICE("eice-first", "vpc-1", "subnet-1", "first.example.com"),
						makeEICE("eice-second", "vpc-1", "subnet-2", "second.example.com"),
					),
					nil,
				)
			},
			wantID: "eice-first",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockEC2 := new(MockEC2API)
			tc.mockSetup(mockEC2)

			client := newTestClient(mockEC2, nil, nil)
			eice, err := client.getEICEByID(tc.eiceID)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, eice)
			assert.Equal(t, tc.wantID, *eice.InstanceConnectEndpointId)
			mockEC2.AssertExpectations(t)
		})
	}
}

func TestClient_CreateEICETunnelURI(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		privateIP  string
		port       string
		eiceID     string
		mockEC2    func(*MockEC2API)
		mockSigner func(*MockHTTPRequestSigner)
		wantURI    string
		wantErr    bool
	}{
		"success": {
			privateIP: "10.0.0.1",
			port:      "22",
			eiceID:    "eice-123",
			mockEC2: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
					makeEICEOutput(makeEICE("eice-123", "vpc-1", "subnet-1", "eice.example.com")),
					nil,
				)
			},
			mockSigner: func(m *MockHTTPRequestSigner) {
				m.On("PresignHTTP", mock.Anything, mock.Anything, mock.Anything,
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
					"wss://eice.example.com/openTunnel?signed=true",
					http.Header{},
					nil,
				)
			},
			wantURI: "wss://eice.example.com/openTunnel?signed=true",
		},
		"different port": {
			privateIP: "10.0.0.1",
			port:      "443",
			eiceID:    "eice-123",
			mockEC2: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
					makeEICEOutput(makeEICE("eice-123", "vpc-1", "subnet-1", "eice.example.com")),
					nil,
				)
			},
			mockSigner: func(m *MockHTTPRequestSigner) {
				m.On("PresignHTTP", mock.Anything, mock.Anything, mock.Anything,
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
					"wss://eice.example.com/openTunnel?remotePort=443&signed=true",
					http.Header{},
					nil,
				)
			},
			wantURI: "wss://eice.example.com/openTunnel?remotePort=443&signed=true",
		},
		"eice not found": {
			privateIP: "10.0.0.1",
			port:      "22",
			eiceID:    "eice-notfound",
			mockEC2: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
					makeEICEOutput(),
					nil,
				)
			},
			mockSigner: func(m *MockHTTPRequestSigner) {
				// Not called because EICE lookup fails
			},
			wantErr: true,
		},
		"signer error": {
			privateIP: "10.0.0.1",
			port:      "22",
			eiceID:    "eice-123",
			mockEC2: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
					makeEICEOutput(makeEICE("eice-123", "vpc-1", "subnet-1", "eice.example.com")),
					nil,
				)
			},
			mockSigner: func(m *MockHTTPRequestSigner) {
				m.On("PresignHTTP", mock.Anything, mock.Anything, mock.Anything,
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
					"",
					http.Header{},
					errors.New("signing failed"),
				)
			},
			wantErr: true,
		},
		"ec2 api error": {
			privateIP: "10.0.0.1",
			port:      "22",
			eiceID:    "eice-123",
			mockEC2: func(m *MockEC2API) {
				m.On("DescribeInstanceConnectEndpoints", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("API error"),
				)
			},
			mockSigner: func(m *MockHTTPRequestSigner) {
				// Not called
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockEC2 := new(MockEC2API)
			mockSigner := new(MockHTTPRequestSigner)
			tc.mockEC2(mockEC2)
			tc.mockSigner(mockSigner)

			client := newTestClient(mockEC2, nil, mockSigner)
			uri, err := client.CreateEICETunnelURI(tc.privateIP, tc.port, tc.eiceID)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantURI, uri)
			mockEC2.AssertExpectations(t)
			mockSigner.AssertExpectations(t)
		})
	}
}

// Note: GuessEICEByVPCAndSubnet uses ec2.NewDescribeInstanceConnectEndpointsPaginator
// which creates a paginator directly from the EC2 client. This is difficult to mock
// without significant restructuring. For comprehensive testing of this function,
// consider integration tests or refactoring to accept an injectable paginator factory.
//
// The function logic is:
// 1. Query all EICE endpoints in the VPC
// 2. Prefer endpoint in same subnet
// 3. Fall back to any endpoint in the VPC
// 4. Return error if no endpoints found
