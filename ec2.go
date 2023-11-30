package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2instanceconnect"
)

var (
	ec2Client                *ec2.EC2
	ec2InstanceConnectClient *ec2instanceconnect.EC2InstanceConnect
)

func ec2init(opts Opts) {
	var config aws.Config

	if opts.region != "" {
		config.Region = aws.String(opts.region)
	} else if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		config.Region = aws.String(region)
	}

	sess := session.Must(session.NewSession(&config))
	ec2Client = ec2.New(sess)
	ec2InstanceConnectClient = ec2instanceconnect.New(sess)
}

func sendSSHPublicKey(instanceID, instanceOSUser, sshPublicKeyPath string) {
	file, err := os.Open(sshPublicKeyPath)
	if err != nil {
		handleError(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	sshPublicKey := scanner.Text()
	if err := scanner.Err(); err != nil {
		handleError(err)
	}

	input := &ec2instanceconnect.SendSSHPublicKeyInput{
		InstanceId:     aws.String(instanceID),
		InstanceOSUser: aws.String(instanceOSUser),
		SSHPublicKey:   aws.String(sshPublicKey),
	}

	_, err = ec2InstanceConnectClient.SendSSHPublicKey(input)
	if err != nil {
		handleError(err)
	}
}

func getInstanceIPByID(instanceID string, usePublicIP bool) string {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	}

	result, err := ec2Client.DescribeInstances(input)
	if err != nil {
		handleError(err)
	}

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if usePublicIP {
				if instance.PublicIpAddress == nil {
					handleError(fmt.Errorf("public IP address not found for instance with ID %s", instanceID))
				}
				return *instance.PublicIpAddress
			} else {
				if instance.PrivateIpAddress == nil {
					handleError(fmt.Errorf("private IP address not found for instance with ID %s", instanceID))
				}
				return *instance.PrivateIpAddress
			}
		}
	}

	handleError(fmt.Errorf("no IP found for instance ID %s", instanceID))
	return ""
}

func getInstanceIDByFilter(filterName, filterValue string) string {
	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(filterName),
				Values: []*string{aws.String(filterValue)},
			},
		},
	}

	result, err := ec2Client.DescribeInstances(input)
	if err != nil {
		handleError(err)
	}

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			return *instance.InstanceId
		}
	}

	handleError(fmt.Errorf("no instance found with %s=%s", filterName, filterValue))
	return ""
}

func guessDestinationType(dst string) DstType {
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
