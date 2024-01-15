package awsutil

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func GetInstanceByID(instanceID string) (types.Instance, error) {
	DebugLogger.Printf("searching for instance by ID %s", instanceID)

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}

	instance, err := getFirstMatchingInstance(input)
	if err != nil {
		return types.Instance{}, fmt.Errorf("unable to find an instance with ID=%s: %w", instanceID, err)
	}

	return instance, nil
}

func GetInstanceByFilter(filterName, filterValue string) (types.Instance, error) {
	DebugLogger.Printf("searching for instance by %s=%s", filterName, filterValue)

	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String(filterName),
				Values: []string{filterValue},
			},
		},
	}

	instance, err := getFirstMatchingInstance(input)
	if err != nil {
		return types.Instance{}, fmt.Errorf("unable to find an instance with %s=%s: %w", filterName, filterValue, err)
	}

	return instance, nil
}

func getFirstMatchingInstance(input *ec2.DescribeInstancesInput) (types.Instance, error) {
	result, err := awsEC2Client.DescribeInstances(context.TODO(), input)
	if err != nil {
		return types.Instance{}, err
	}

	DebugLogger.Printf("found %d reservations", len(result.Reservations))

	for rsvIdx, reservation := range result.Reservations {
		DebugLogger.Printf("found %d instances in reservation %d", len(reservation.Instances), rsvIdx)

		for _, instance := range reservation.Instances {
			DebugLogger.Printf("selected first matching instance %s", *instance.InstanceId)

			return instance, nil
		}
	}

	return types.Instance{}, fmt.Errorf("%w in %s", ErrNoMatches, awsRegion)
}
