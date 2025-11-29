package ec2

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	signerV4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
)

// Client wraps AWS SDK clients for EC2 and EC2 Instance Connect operations.
type Client struct {
	ec2Client     *ec2.Client
	connectClient *ec2instanceconnect.Client
	signer        *signerV4.Signer
	credentials   aws.Credentials
	region        string
	logger        *log.Logger
}

// NewClient creates a new Client with the given region, profile, and logger.
func NewClient(region, profile string, logger *log.Logger) (*Client, error) {
	optFns := make([]func(*config.LoadOptions) error, 0)

	if region != "" {
		logger.Printf("using region %s", region)
		optFns = append(optFns, config.WithRegion(region))
	}

	if profile != "" {
		logger.Printf("using profile %s", profile)
		optFns = append(optFns, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), optFns...)
	if err != nil {
		return nil, err
	}

	// Credentials and region are required for Signer API
	credentials, err := cfg.Credentials.Retrieve(context.TODO())
	if err != nil {
		return nil, err
	}

	return &Client{
		ec2Client:     ec2.NewFromConfig(cfg),
		connectClient: ec2instanceconnect.NewFromConfig(cfg),
		signer:        signerV4.NewSigner(),
		credentials:   credentials,
		region:        cfg.Region,
		logger:        logger,
	}, nil
}

// Region returns the AWS region this client is configured for.
func (c *Client) Region() string {
	return c.region
}
