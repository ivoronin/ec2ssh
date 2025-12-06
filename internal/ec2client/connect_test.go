package ec2client

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestClient_SendSSHPublicKey(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		instanceID  string
		osUser      string
		publicKey   string
		mockSetup   func(*MockEC2InstanceConnectAPI)
		wantErr     bool
		errContains string
	}{
		"success": {
			instanceID: "i-1234567890abcdef0",
			osUser:     "ec2-user",
			publicKey:  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample user@host",
			mockSetup: func(m *MockEC2InstanceConnectAPI) {
				m.On("SendSSHPublicKey", mock.Anything, mock.MatchedBy(func(input *ec2instanceconnect.SendSSHPublicKeyInput) bool {
					return *input.InstanceId == "i-1234567890abcdef0" &&
						*input.InstanceOSUser == "ec2-user" &&
						*input.SSHPublicKey == "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample user@host"
				})).Return(&ec2instanceconnect.SendSSHPublicKeyOutput{}, nil)
			},
		},
		"success with different user": {
			instanceID: "i-abc123",
			osUser:     "ubuntu",
			publicKey:  "ssh-rsa AAAAB3... user@host",
			mockSetup: func(m *MockEC2InstanceConnectAPI) {
				m.On("SendSSHPublicKey", mock.Anything, mock.MatchedBy(func(input *ec2instanceconnect.SendSSHPublicKeyInput) bool {
					return *input.InstanceId == "i-abc123" &&
						*input.InstanceOSUser == "ubuntu"
				})).Return(&ec2instanceconnect.SendSSHPublicKeyOutput{}, nil)
			},
		},
		"api error - access denied": {
			instanceID: "i-1234567890abcdef0",
			osUser:     "ec2-user",
			publicKey:  "ssh-ed25519 AAAAC3...",
			mockSetup: func(m *MockEC2InstanceConnectAPI) {
				m.On("SendSSHPublicKey", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("AccessDeniedException: User is not authorized"),
				)
			},
			wantErr:     true,
			errContains: "AccessDeniedException",
		},
		"api error - instance not found": {
			instanceID: "i-notfound",
			osUser:     "ec2-user",
			publicKey:  "ssh-ed25519 AAAAC3...",
			mockSetup: func(m *MockEC2InstanceConnectAPI) {
				m.On("SendSSHPublicKey", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("InvalidInstanceId: The instance ID 'i-notfound' is not valid"),
				)
			},
			wantErr:     true,
			errContains: "InvalidInstanceId",
		},
		"api error - throttling": {
			instanceID: "i-throttled",
			osUser:     "ec2-user",
			publicKey:  "ssh-ed25519 AAAAC3...",
			mockSetup: func(m *MockEC2InstanceConnectAPI) {
				m.On("SendSSHPublicKey", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("ThrottlingException: Rate exceeded"),
				)
			},
			wantErr:     true,
			errContains: "ThrottlingException",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockConnect := new(MockEC2InstanceConnectAPI)
			tc.mockSetup(mockConnect)

			instance := makeInstance(tc.instanceID)
			client := newTestClient(nil, mockConnect, nil)

			err := client.SendSSHPublicKey(instance, tc.osUser, tc.publicKey)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			mockConnect.AssertExpectations(t)
		})
	}
}
