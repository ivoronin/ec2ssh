package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
)

var (
	ec2Client                *ec2.Client
	ec2InstanceConnectClient *ec2instanceconnect.Client
)

func AWSInit(opts Opts) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)

	if opts.region != "" {
		cfg.Region = opts.region
	}

	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	ec2Client = ec2.NewFromConfig(cfg)
	ec2InstanceConnectClient = ec2instanceconnect.NewFromConfig(cfg)
}
