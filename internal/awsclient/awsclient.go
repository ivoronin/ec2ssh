// Package awsclient provides AWS SDK configuration loading.
package awsclient

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// LoadConfig loads AWS SDK configuration with optional region and profile.
func LoadConfig(region, profile string, logger *log.Logger) (aws.Config, error) {
	optFns := make([]func(*config.LoadOptions) error, 0)

	if region != "" {
		if logger != nil {
			logger.Printf("using region %s", region)
		}
		optFns = append(optFns, config.WithRegion(region))
	}

	if profile != "" {
		if logger != nil {
			logger.Printf("using profile %s", profile)
		}
		optFns = append(optFns, config.WithSharedConfigProfile(profile))
	}

	return config.LoadDefaultConfig(context.TODO(), optFns...)
}
