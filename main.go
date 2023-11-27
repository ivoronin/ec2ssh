package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
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

func getECInstanceIPByID(instanceID string) string {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	}

	result, err := ec2Client.DescribeInstances(input)
	if err != nil {
		handleError(err)
	}

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			return *instance.PrivateIpAddress
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

func getSSHPublicKey() string {
	sshPublicKeyPath := os.Getenv(("SSH_PUBLIC_KEY_PATH"))
	if sshPublicKeyPath == "" {
		usr, err := user.Current()
		if err != nil {
			handleError(err)
		}
		sshPublicKeyPath = usr.HomeDir + "/.ssh/id_rsa.pub"
	}

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

func handleError(err error) {
	fmt.Fprintf(os.Stderr, "An error occurred: %v\n", err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: ec2ssh [-l login_user] [other ssh flags] destination [command [argument ...]]\n")
	os.Exit(1)
}

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		usage()
	}

	sshArgs := make([]string, len(args))
	copy(sshArgs, args)

	loginUser := "ec2-user"
	var destination string
	var destinationIndex int

	for i := 0; i < len(args); i++ {
		if args[i] == "-l" && i+1 < len(args) {
			loginUser = args[i+1]
			i++ // Skip next argument
		} else if !strings.HasPrefix(args[i], "-") {
			if destination == "" {
				destinationIndex = i
				destination = args[i]
			}
		}
	}

	if destination == "" {
		usage()
	}

	var instanceID string
	destinationIP := net.ParseIP(destination)
	if destinationIP != nil {
		if destinationIP.IsPrivate() {
			instanceID = getEC2InstanceIDByFilter("private-ip-address", destination)
		} else {
			instanceID = getEC2InstanceIDByFilter("ip-address", destination)
		}
	} else if strings.HasPrefix(destination, "i-") {
		instanceID = destination
	} else if strings.HasPrefix(destination, "ip-") {
		instanceID = getEC2InstanceIDByFilter("private-dns-name", destination+".*")
	} else {
		instanceID = getEC2InstanceIDByFilter("tag:Name", destination)
	}

	sshDestination := getECInstanceIPByID(instanceID)
	sshPublicKey := getSSHPublicKey()

	sendSSHPublicKey(instanceID, loginUser, sshPublicKey)

	sshArgs[destinationIndex] = sshDestination
	cmd := exec.Command("ssh", sshArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		handleError(err)
	}
}
