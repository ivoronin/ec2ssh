package main

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func GetInstanceById(instanceId string) *types.Instance {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceId},
	}
	instance := getFirstMatchingInstance(input)
	if instance == nil {
		HandleError(fmt.Errorf("no instance found with id %s", instanceId))
	}
	return instance
}

func GetInstanceByFilter(filterName, filterValue string) *types.Instance {
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
		HandleError(fmt.Errorf("no instance found with %s=%s", filterName, filterValue))
	}
	return instance
}

func getFirstMatchingInstance(input *ec2.DescribeInstancesInput) *types.Instance {
	ctx := context.Background()
	result, err := ec2Client.DescribeInstances(ctx, input)
	if err != nil {
		HandleError(err)
	}

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			return &instance
		}
	}

	return nil
}

func GuessDestinationType(dst string) DstType {
	if strings.HasPrefix(dst, "i-") {
		return DstTypeID
	}

	if strings.HasPrefix(dst, "ip-") {
		return DstTypePrivateDNSName
	}

	ip := net.ParseIP(dst)
	if ip != nil {
		if ip.IsPrivate() {
			return DstTypePrivateIP
		} else {
			return DstTypePublicIP
		}
	}

	return DstTypeNameTag
}
