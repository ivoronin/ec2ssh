package ec2

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSendSSHPublicKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		instance   func() makeInstanceResult
		osUser     string
		publicKey  string
		setupMock  func(*MockEC2InstanceConnectAPI)
		wantErr    bool
		errContain string
	}{
		{
			name: "success",
			instance: func() makeInstanceResult {
				return makeInstanceResult{makeInstance("i-0123456789abcdef0")}
			},
			osUser:    "ec2-user",
			publicKey: "ssh-ed25519 AAAA... test@host",
			setupMock: func(m *MockEC2InstanceConnectAPI) {
				m.On("SendSSHPublicKey", mock.Anything, mock.MatchedBy(func(input *ec2instanceconnect.SendSSHPublicKeyInput) bool {
					return *input.InstanceId == "i-0123456789abcdef0" &&
						*input.InstanceOSUser == "ec2-user" &&
						*input.SSHPublicKey == "ssh-ed25519 AAAA... test@host"
				})).Return(&ec2instanceconnect.SendSSHPublicKeyOutput{Success: true}, nil)
			},
			wantErr: false,
		},
		{
			name: "error - invalid key",
			instance: func() makeInstanceResult {
				return makeInstanceResult{makeInstance("i-test")}
			},
			osUser:    "ubuntu",
			publicKey: "invalid-key",
			setupMock: func(m *MockEC2InstanceConnectAPI) {
				m.On("SendSSHPublicKey", mock.Anything, mock.Anything).Return(
					nil, errors.New("InvalidArgsException: SSH public key is not valid"))
			},
			wantErr:    true,
			errContain: "InvalidArgsException",
		},
		{
			name: "error - instance not found",
			instance: func() makeInstanceResult {
				return makeInstanceResult{makeInstance("i-nonexistent")}
			},
			osUser:    "ec2-user",
			publicKey: "ssh-ed25519 AAAA...",
			setupMock: func(m *MockEC2InstanceConnectAPI) {
				m.On("SendSSHPublicKey", mock.Anything, mock.Anything).Return(
					nil, errors.New("NotFoundException: Instance not found"))
			},
			wantErr:    true,
			errContain: "NotFoundException",
		},
		{
			name: "error - rate limited",
			instance: func() makeInstanceResult {
				return makeInstanceResult{makeInstance("i-test")}
			},
			osUser:    "ec2-user",
			publicKey: "ssh-ed25519 AAAA...",
			setupMock: func(m *MockEC2InstanceConnectAPI) {
				m.On("SendSSHPublicKey", mock.Anything, mock.Anything).Return(
					nil, errors.New("ThrottlingException: Rate exceeded"))
			},
			wantErr:    true,
			errContain: "ThrottlingException",
		},
		{
			name: "error - permission denied",
			instance: func() makeInstanceResult {
				return makeInstanceResult{makeInstance("i-test")}
			},
			osUser:    "root",
			publicKey: "ssh-ed25519 AAAA...",
			setupMock: func(m *MockEC2InstanceConnectAPI) {
				m.On("SendSSHPublicKey", mock.Anything, mock.Anything).Return(
					nil, errors.New("AccessDeniedException: User not authorized"))
			},
			wantErr:    true,
			errContain: "AccessDeniedException",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockConnect := new(MockEC2InstanceConnectAPI)
			tt.setupMock(mockConnect)
			client := newTestClient(nil, mockConnect, nil)

			instanceResult := tt.instance()
			err := client.SendSSHPublicKey(instanceResult.Instance, tt.osUser, tt.publicKey)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				mockConnect.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			mockConnect.AssertExpectations(t)
		})
	}
}

// makeInstanceResult wraps an instance for cleaner test table syntax.
type makeInstanceResult struct {
	Instance types.Instance
}
