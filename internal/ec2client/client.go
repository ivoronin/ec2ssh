package ec2client

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	signerV4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
)

// Client wraps AWS SDK clients for EC2 and EC2 Instance Connect operations.
type Client struct {
	ec2Client     EC2API
	connectClient EC2InstanceConnectAPI
	signer        HTTPRequestSigner
	credentials   aws.Credentials
	region        string
	logger        *log.Logger
}

// NewClient creates a new Client from an existing AWS config.
func NewClient(cfg aws.Config, logger *log.Logger) (*Client, error) {
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
