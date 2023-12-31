package awsutil

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func GetInstanceByID(instanceID string) (*types.Instance, error) {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}

	instance := getFirstMatchingInstance(input)
	if instance == nil {
		return nil, fmt.Errorf("%w: no instance found with id %s", ErrNotFound, instanceID)
	}

	return instance, nil
}

func GetInstanceByFilter(filterName, filterValue string) (*types.Instance, error) {
	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String(filterName),
				Values: []string{filterValue},
			},
		},
	}

	instance := getFirstMatchingInstance(input)
	if instance == nil {
		return nil, fmt.Errorf("%w: no instance found with %s=%s", ErrNotFound, filterName, filterValue)
	}

	return instance, nil
}

func getFirstMatchingInstance(input *ec2.DescribeInstancesInput) *types.Instance {
	result, err := awsEC2Client.DescribeInstances(context.TODO(), input)
	if err != nil {
		return nil
	}

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			return &instance
		}
	}

	return nil
}
