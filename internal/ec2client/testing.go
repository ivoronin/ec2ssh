package ec2client

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	signerV4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
	"github.com/stretchr/testify/mock"
)

// =============================================================================
// Mock Implementations - Exported for use by other packages' tests
// =============================================================================

// MockEC2API is a mock implementation of EC2API.
type MockEC2API struct {
	mock.Mock
}

// DescribeInstances mocks the EC2 DescribeInstances API call.
func (m *MockEC2API) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ec2.DescribeInstancesOutput), args.Error(1)
}

// DescribeInstanceConnectEndpoints mocks the EC2 DescribeInstanceConnectEndpoints API call.
func (m *MockEC2API) DescribeInstanceConnectEndpoints(ctx context.Context, params *ec2.DescribeInstanceConnectEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceConnectEndpointsOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ec2.DescribeInstanceConnectEndpointsOutput), args.Error(1)
}

// MockEC2InstanceConnectAPI is a mock implementation of EC2InstanceConnectAPI.
type MockEC2InstanceConnectAPI struct {
	mock.Mock
}

// SendSSHPublicKey mocks the EC2 Instance Connect SendSSHPublicKey API call.
func (m *MockEC2InstanceConnectAPI) SendSSHPublicKey(ctx context.Context, params *ec2instanceconnect.SendSSHPublicKeyInput, optFns ...func(*ec2instanceconnect.Options)) (*ec2instanceconnect.SendSSHPublicKeyOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ec2instanceconnect.SendSSHPublicKeyOutput), args.Error(1)
}

// MockHTTPRequestSigner is a mock implementation of HTTPRequestSigner.
type MockHTTPRequestSigner struct {
	mock.Mock
}

// PresignHTTP mocks the HTTP request signing.
func (m *MockHTTPRequestSigner) PresignHTTP(ctx context.Context, credentials aws.Credentials, r *http.Request, payloadHash string, service string, region string, signingTime time.Time, optFns ...func(*signerV4.SignerOptions)) (string, http.Header, error) {
	args := m.Called(ctx, credentials, r, payloadHash, service, region, signingTime)
	var headers http.Header
	if args.Get(1) != nil {
		headers = args.Get(1).(http.Header)
	}
	return args.String(0), headers, args.Error(2)
}

// =============================================================================
// Test Client Factory
// =============================================================================

// NewTestClient creates a Client with mock dependencies for testing.
// This is intended for use by test code in other packages.
func NewTestClient(ec2API EC2API, connectAPI EC2InstanceConnectAPI, signer HTTPRequestSigner) *Client {
	return &Client{
		ec2Client:     ec2API,
		connectClient: connectAPI,
		signer:        signer,
		credentials:   aws.Credentials{},
		region:        "us-east-1",
		logger:        log.New(io.Discard, "", 0),
	}
}

// =============================================================================
// Test Instance Builders - Exported for use by other packages' tests
// =============================================================================

// MakeInstance creates a test instance with the given ID and options.
func MakeInstance(id string, opts ...func(*types.Instance)) types.Instance {
	inst := types.Instance{
		InstanceId: aws.String(id),
		State:      &types.InstanceState{Name: types.InstanceStateNameRunning},
	}
	for _, opt := range opts {
		opt(&inst)
	}
	return inst
}

// WithPrivateIP sets the private IP address.
func WithPrivateIP(ip string) func(*types.Instance) {
	return func(i *types.Instance) {
		i.PrivateIpAddress = aws.String(ip)
	}
}

// WithPublicIP sets the public IP address.
func WithPublicIP(ip string) func(*types.Instance) {
	return func(i *types.Instance) {
		i.PublicIpAddress = aws.String(ip)
	}
}

// WithIPv6 sets the IPv6 address.
func WithIPv6(ip string) func(*types.Instance) {
	return func(i *types.Instance) {
		i.Ipv6Address = aws.String(ip)
	}
}

// WithVPC sets the VPC ID.
func WithVPC(vpcID string) func(*types.Instance) {
	return func(i *types.Instance) {
		i.VpcId = aws.String(vpcID)
	}
}

// WithSubnet sets the subnet ID.
func WithSubnet(subnetID string) func(*types.Instance) {
	return func(i *types.Instance) {
		i.SubnetId = aws.String(subnetID)
	}
}

// WithNameTag adds a Name tag.
func WithNameTag(name string) func(*types.Instance) {
	return func(i *types.Instance) {
		i.Tags = append(i.Tags, types.Tag{
			Key:   aws.String("Name"),
			Value: aws.String(name),
		})
	}
}

// WithTag adds a custom tag.
func WithTag(key, value string) func(*types.Instance) {
	return func(i *types.Instance) {
		i.Tags = append(i.Tags, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
}

// =============================================================================
// Test Output Builders - Exported for use by other packages' tests
// =============================================================================

// MakeReservation wraps instances in a reservation.
func MakeReservation(instances ...types.Instance) types.Reservation {
	return types.Reservation{Instances: instances}
}

// MakeDescribeOutput creates DescribeInstancesOutput with given reservations.
func MakeDescribeOutput(reservations ...types.Reservation) *ec2.DescribeInstancesOutput {
	return &ec2.DescribeInstancesOutput{Reservations: reservations}
}

// MakeEICE creates a test EC2 Instance Connect Endpoint.
func MakeEICE(id, vpcID, subnetID, dnsName string) types.Ec2InstanceConnectEndpoint {
	return types.Ec2InstanceConnectEndpoint{
		InstanceConnectEndpointId: aws.String(id),
		VpcId:                     aws.String(vpcID),
		SubnetId:                  aws.String(subnetID),
		DnsName:                   aws.String(dnsName),
	}
}

// MakeEICEOutput creates DescribeInstanceConnectEndpointsOutput.
func MakeEICEOutput(endpoints ...types.Ec2InstanceConnectEndpoint) *ec2.DescribeInstanceConnectEndpointsOutput {
	return &ec2.DescribeInstanceConnectEndpointsOutput{
		InstanceConnectEndpoints: endpoints,
	}
}

// =============================================================================
// Type Pointer Helpers - Exported for use by other packages' tests
// =============================================================================

// AddrTypePtr returns a pointer to the AddrType value.
func AddrTypePtr(t AddrType) *AddrType {
	return &t
}

// DstTypePtr returns a pointer to the DstType value.
func DstTypePtr(t DstType) *DstType {
	return &t
}
