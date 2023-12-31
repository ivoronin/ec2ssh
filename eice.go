package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
)

func SendSSHPublicKey(instanceID, instanceOSUser, sshPublicKeyPath string) {
	ctx := context.Background()
	file, err := os.Open(sshPublicKeyPath)
	if err != nil {
		HandleError(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	sshPublicKey := scanner.Text()
	if err := scanner.Err(); err != nil {
		HandleError(err)
	}

	input := &ec2instanceconnect.SendSSHPublicKeyInput{
		InstanceId:     aws.String(instanceID),
		InstanceOSUser: aws.String(instanceOSUser),
		SSHPublicKey:   aws.String(sshPublicKey),
	}

	_, err = ec2InstanceConnectClient.SendSSHPublicKey(ctx, input)
	if err != nil {
		HandleError(err)
	}
}

func GetInstanceConnectEndpointByID(instanceConnectEndpointID string) *ec2Types.Ec2InstanceConnectEndpoint {
	ctx := context.Background()

	input := &ec2.DescribeInstanceConnectEndpointsInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{"create-complete"},
			},
		},
		InstanceConnectEndpointIds: []string{instanceConnectEndpointID},
	}

	result, err := ec2Client.DescribeInstanceConnectEndpoints(ctx, input)
	if err != nil {
		HandleError(err)
	}

	if len(result.InstanceConnectEndpoints) > 0 {
		return &result.InstanceConnectEndpoints[0]
	}

	HandleError(fmt.Errorf("no instance connect endpoint found with ID %s", instanceConnectEndpointID))
	return nil
}

func GetInstanceConnectEndpointByVpc(vpcID string, subnetID string) *ec2Types.Ec2InstanceConnectEndpoint {
	ctx := context.Background()
	input := &ec2.DescribeInstanceConnectEndpointsInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{"create-complete"},
			},
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	}

	var instanceConnectEndpoints []ec2Types.Ec2InstanceConnectEndpoint

	// Using a paginator to handle potentially paginated results
	paginator := ec2.NewDescribeInstanceConnectEndpointsPaginator(ec2Client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			HandleError(err)
		}

		for _, eice := range page.InstanceConnectEndpoints {
			instanceConnectEndpoints = append(instanceConnectEndpoints, eice)
			if *eice.SubnetId == subnetID {
				return &eice
			}
		}
	}

	if len(instanceConnectEndpoints) > 0 {
		return &instanceConnectEndpoints[0]
	}

	HandleError(fmt.Errorf("no instance connect endpoint found for for %s/%s", vpcID, subnetID))
	return nil
}
