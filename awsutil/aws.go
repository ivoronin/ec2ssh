package awsutil

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	signerV4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
)

var ErrNoMatches = fmt.Errorf("no matches")

var (
	awsEC2Client                *ec2.Client
	awsEC2InstanceConnectClient *ec2instanceconnect.Client
	awsSigner                   *signerV4.Signer
	awsCredentials              aws.Credentials
	awsRegion                   string
)

func Init(region string, profile string) (err error) {
	optFns := make([]func(*config.LoadOptions) error, 0)

	if region != "" {
		optFns = append(optFns, config.WithRegion(region))
	}

	if profile != "" {
		optFns = append(optFns, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), optFns...)
	if err != nil {
		return err
	}

	/* Credentials and region are required for Signer API */
	awsCredentials, err = cfg.Credentials.Retrieve(context.TODO())
	if err != nil {
		return err
	}

	awsRegion = cfg.Region
	awsEC2Client = ec2.NewFromConfig(cfg)
	awsEC2InstanceConnectClient = ec2instanceconnect.NewFromConfig(cfg)
	awsSigner = signerV4.NewSigner()

	return nil
}
