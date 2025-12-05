package ec2client

import (
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	signerV4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
)

// EC2API abstracts the AWS EC2 API operations used by this package.
// This interface enables testing with mock implementations.
type EC2API interface {
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeInstanceConnectEndpoints(ctx context.Context, params *ec2.DescribeInstanceConnectEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceConnectEndpointsOutput, error)
}

// EC2InstanceConnectAPI abstracts the EC2 Instance Connect API operations.
type EC2InstanceConnectAPI interface {
	SendSSHPublicKey(ctx context.Context, params *ec2instanceconnect.SendSSHPublicKeyInput, optFns ...func(*ec2instanceconnect.Options)) (*ec2instanceconnect.SendSSHPublicKeyOutput, error)
}

// HTTPRequestSigner abstracts the HTTP request signing operations.
type HTTPRequestSigner interface {
	PresignHTTP(ctx context.Context, credentials aws.Credentials, r *http.Request, payloadHash string, service string, region string, signingTime time.Time, optFns ...func(*signerV4.SignerOptions)) (signedURI string, signedHeaders http.Header, err error)
}

// Ensure AWS SDK clients implement our interfaces at compile time.
var (
	_ EC2API                 = (*ec2.Client)(nil)
	_ EC2InstanceConnectAPI  = (*ec2instanceconnect.Client)(nil)
	_ HTTPRequestSigner      = (*signerV4.Signer)(nil)
)
