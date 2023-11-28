package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
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

func init() {
	var config aws.Config
	region := os.Getenv("AWS_DEFAULT_REGION")
	if region != "" {
		config.Region = aws.String(region)
	}
	sess := session.Must(session.NewSession(&config))
	ec2Client = ec2.New(sess)
	ec2InstanceConnectClient = ec2instanceconnect.New(sess)
}

func sendSSHPublicKey(instanceID, instanceOSUser, sshPublicKey string) {
	input := &ec2instanceconnect.SendSSHPublicKeyInput{
		InstanceId:     aws.String(instanceID),
		InstanceOSUser: aws.String(instanceOSUser),
		SSHPublicKey:   aws.String(sshPublicKey),
	}

	_, err := ec2InstanceConnectClient.SendSSHPublicKey(input)
	if err != nil {
		handleError(err)
	}
}

func getEC2InstanceIPByID(instanceID string, usePublicIP bool) string {
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

func getEC2InstanceIDByFilter(filterName, filterValue string) string {
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

func getSSHPublicKey(sshPublicKeyPath string) string {
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
	return sshPublicKey
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

func handleError(err error) {
	fmt.Fprintf(os.Stderr, "ec2ssh: error: %v\n", err)
	os.Exit(1)
}

func handleWarning(msg string) {
	fmt.Fprintf(os.Stderr, "ec2ssh: warning: %s\n", msg)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: ec2ssh [--ssh-public-key path] [--use-public-ip]\n")
	fmt.Fprintf(os.Stderr, "        [--destination-type <id|private_ip|public_ip|private_dns|name_tag>]\n")
	fmt.Fprintf(os.Stderr, "        [-l login_user] [other ssh flags] destination [command [argument ...]]\n")
	os.Exit(1)
}

func main() {
	opts, sshArgs := parseArgs()

	dstType := opts.dstType
	if dstType == DstTypeUnknown {
		dstType = guessDestinationType(sshArgs.Destination())
	}

	var instanceID string
	switch dstType {
	case DstTypeID:
		instanceID = sshArgs.Destination()
	case DstTypePrivateIP:
		instanceID = getEC2InstanceIDByFilter("private-ip-address", sshArgs.Destination())
	case DstTypePublicIP:
		instanceID = getEC2InstanceIDByFilter("ip-address", sshArgs.Destination())
	case DstTypePrivateDNSName:
		instanceID = getEC2InstanceIDByFilter("private-dns-name", sshArgs.Destination()+".*")
	case DstTypeNameTag:
		instanceID = getEC2InstanceIDByFilter("tag:Name", sshArgs.Destination())
	}

	if dstType != DstTypePrivateIP && dstType != DstTypePublicIP {
		ip := getEC2InstanceIPByID(instanceID, opts.usePublicIP)
		sshArgs.SetDestination(ip)
	} else if opts.usePublicIP {
		handleWarning("the option '--use-public-ip' is ignored since an IP address has been provided")
	}

	sshPublicKey := getSSHPublicKey(opts.sshPublicKeyPath)
	sendSSHPublicKey(instanceID, opts.loginUser, sshPublicKey)

	cmd := exec.Command("ssh", sshArgs.Args()...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		handleError(err)
	}
}
