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

// MockEC2API is a mock implementation of EC2API for testing.
type MockEC2API struct {
	mock.Mock
}

func (m *MockEC2API) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ec2.DescribeInstancesOutput), args.Error(1)
}

func (m *MockEC2API) DescribeInstanceConnectEndpoints(ctx context.Context, params *ec2.DescribeInstanceConnectEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceConnectEndpointsOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ec2.DescribeInstanceConnectEndpointsOutput), args.Error(1)
}

// MockEC2InstanceConnectAPI is a mock implementation of EC2InstanceConnectAPI for testing.
type MockEC2InstanceConnectAPI struct {
	mock.Mock
}

func (m *MockEC2InstanceConnectAPI) SendSSHPublicKey(ctx context.Context, params *ec2instanceconnect.SendSSHPublicKeyInput, optFns ...func(*ec2instanceconnect.Options)) (*ec2instanceconnect.SendSSHPublicKeyOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ec2instanceconnect.SendSSHPublicKeyOutput), args.Error(1)
}

// MockHTTPSigner is a mock implementation of the HTTP request signer.
type MockHTTPSigner struct {
	mock.Mock
}

func (m *MockHTTPSigner) PresignHTTP(ctx context.Context, credentials aws.Credentials, r *http.Request, payloadHash string, service string, region string, signingTime time.Time, optFns ...func(*signerV4.SignerOptions)) (string, http.Header, error) {
	args := m.Called(ctx, credentials, r, payloadHash, service, region, signingTime)
	var headers http.Header
	if args.Get(1) != nil {
		headers = args.Get(1).(http.Header)
	}
	return args.String(0), headers, args.Error(2)
}

// defaultMockSigner is a simple signer that captures the URL and adds a mock signature
type defaultMockSigner struct{}

func (d *defaultMockSigner) PresignHTTP(ctx context.Context, credentials aws.Credentials, r *http.Request, payloadHash string, service string, region string, signingTime time.Time, optFns ...func(*signerV4.SignerOptions)) (string, http.Header, error) {
	return r.URL.String() + "&X-Amz-Signature=mocksignature", nil, nil
}

// newTestClient creates a Client with mock dependencies for testing.
func newTestClient(ec2API EC2API, connectAPI EC2InstanceConnectAPI, signer HTTPRequestSigner) *Client {
	// Create a default mock signer if none provided
	if signer == nil {
		signer = &defaultMockSigner{}
	}
	return &Client{
		ec2Client:     ec2API,
		connectClient: connectAPI,
		signer:        signer,
		credentials:   aws.Credentials{AccessKeyID: "AKIATEST", SecretAccessKey: "secret", SessionToken: "token"},
		region:        "us-east-1",
		logger:        log.New(io.Discard, "", 0),
	}
}

// Test fixture helpers

// makeInstance creates a test EC2 instance with the given ID and options.
func makeInstance(id string, opts ...func(*types.Instance)) types.Instance {
	instance := types.Instance{
		InstanceId: aws.String(id),
	}
	for _, opt := range opts {
		opt(&instance)
	}
	return instance
}

// withPrivateIP adds a private IP to the instance.
func withPrivateIP(ip string) func(*types.Instance) {
	return func(i *types.Instance) {
		i.PrivateIpAddress = aws.String(ip)
	}
}

// withPublicIP adds a public IP to the instance.
func withPublicIP(ip string) func(*types.Instance) {
	return func(i *types.Instance) {
		i.PublicIpAddress = aws.String(ip)
	}
}

// withIPv6 adds an IPv6 address to the instance.
func withIPv6(ip string) func(*types.Instance) {
	return func(i *types.Instance) {
		i.Ipv6Address = aws.String(ip)
	}
}

// withVPC adds VPC and subnet IDs to the instance.
func withVPC(vpcID, subnetID string) func(*types.Instance) {
	return func(i *types.Instance) {
		i.VpcId = aws.String(vpcID)
		i.SubnetId = aws.String(subnetID)
	}
}

// withNameTag adds a Name tag to the instance.
func withNameTag(name string) func(*types.Instance) {
	return func(i *types.Instance) {
		i.Tags = append(i.Tags, types.Tag{
			Key:   aws.String("Name"),
			Value: aws.String(name),
		})
	}
}

// makeEICE creates a test EC2 Instance Connect Endpoint.
func makeEICE(id, dnsName, vpcID, subnetID string) types.Ec2InstanceConnectEndpoint {
	return types.Ec2InstanceConnectEndpoint{
		InstanceConnectEndpointId: aws.String(id),
		DnsName:                   aws.String(dnsName),
		VpcId:                     aws.String(vpcID),
		SubnetId:                  aws.String(subnetID),
		State:                     types.Ec2InstanceConnectEndpointStateCreateComplete,
	}
}

// describeInstancesOutput creates a DescribeInstancesOutput from instances.
func describeInstancesOutput(instances ...types.Instance) *ec2.DescribeInstancesOutput {
	if len(instances) == 0 {
		return &ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{},
		}
	}
	return &ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{
			{Instances: instances},
		},
	}
}

// describeEICEOutput creates a DescribeInstanceConnectEndpointsOutput.
func describeEICEOutput(endpoints ...types.Ec2InstanceConnectEndpoint) *ec2.DescribeInstanceConnectEndpointsOutput {
	return &ec2.DescribeInstanceConnectEndpointsOutput{
		InstanceConnectEndpoints: endpoints,
	}
}
