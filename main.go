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

type DestinationType int

const (
	Unknown DestinationType = iota
	ID
	PrivateIP
	PublicIP
	NameTag
	PrivateDNSName
)

type Opts struct {
	loginUser        string
	sshPublicKeyPath string
	usePublicIP      bool
	destinationType  DestinationType
	sshArgs          []string
	destinationIdx   int
	destination      string
}

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

func getECInstanceIPByID(instanceID string, usePublicIP bool) string {
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

func guessDestinationType(destination string) DestinationType {
	if strings.HasPrefix(destination, "i-") {
		return ID
	}

	if strings.HasPrefix(destination, "ip-") {
		return PrivateDNSName
	}

	ip := net.ParseIP(destination)
	if ip != nil {
		if ip.IsPrivate() {
			return PrivateIP
		} else {
			return PublicIP
		}
	}

	return NameTag
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

func parseArgs() Opts {
	args := os.Args[1:]
	if len(args) < 1 {
		usage()
	}

	usr, err := user.Current()
	if err != nil {
		handleError(err)
	}

	/* default values */
	opts := Opts{
		loginUser:        "ec2-user",
		sshPublicKeyPath: usr.HomeDir + "/.ssh/id_rsa.pub",
		usePublicIP:      false,
		destinationType:  Unknown,
		sshArgs:          make([]string, 0, len(args)),
	}

	for i := 0; i < len(args); i++ {
		/* ssh doesn't use long keys */
		if strings.HasPrefix(args[i], "--") && len(args[i]) > 2 {
			switch args[i] {
			case "--public-key":
				if i+1 >= len(args) {
					handleError(fmt.Errorf("public key path not provided"))
				}
				opts.sshPublicKeyPath = args[i+1]
				i++
			case "--use-public-ip":
				opts.usePublicIP = true
			case "--destination-type":
				if i+1 >= len(args) {
					handleError(fmt.Errorf("destination type not provided"))
				}
				switch args[i+1] {
				case "id":
					opts.destinationType = ID
				case "private_ip":
					opts.destinationType = PrivateIP
				case "public_ip":
					opts.destinationType = PublicIP
				case "private_dns":
					opts.destinationType = PrivateDNSName
				case "name_tag":
					opts.destinationType = NameTag
				default:
					handleError(fmt.Errorf("unknown destination type: %s", args[i+1]))
				}
				i++
			default:
				handleError(fmt.Errorf("unknown option %s", args[i]))
			}
			continue
		}

		opts.sshArgs = append(opts.sshArgs, args[i])
		if args[i] == "-l" && i+1 < len(args) {
			opts.loginUser = args[i+1]
			// Skip next argument
			i++
			opts.sshArgs = append(opts.sshArgs, args[i])
		} else if !strings.HasPrefix(args[i], "-") {
			if opts.destination == "" {
				opts.destinationIdx = len(opts.sshArgs) - 1
				opts.destination = args[i]
			}
		}
	}

	if opts.destination == "" {
		usage()
	}

	return opts
}

func main() {
	opts := parseArgs()

	destinationType := opts.destinationType
	if destinationType == Unknown {
		destinationType = guessDestinationType(opts.destination)
	}

	var instanceID string
	switch destinationType {
	case ID:
		instanceID = opts.destination
	case PrivateIP:
		instanceID = getEC2InstanceIDByFilter("private-ip-address", opts.destination)
	case PublicIP:
		instanceID = getEC2InstanceIDByFilter("ip-address", opts.destination)
	case PrivateDNSName:
		instanceID = getEC2InstanceIDByFilter("private-dns-name", opts.destination+".*")
	case NameTag:
		instanceID = getEC2InstanceIDByFilter("tag:Name", opts.destination)
	}

	var sshDestination string
	switch destinationType {
	case PrivateIP, PublicIP:
		if opts.usePublicIP {
			handleWarning("the option '--use-public-ip' is ignored since an IP address has been provided")
		}
		sshDestination = opts.destination
	default:
		sshDestination = getECInstanceIPByID(instanceID, opts.usePublicIP)
	}

	sshPublicKey := getSSHPublicKey(opts.sshPublicKeyPath)
	sendSSHPublicKey(instanceID, opts.loginUser, sshPublicKey)

	sshArgs := make([]string, len(opts.sshArgs))
	copy(sshArgs, opts.sshArgs)
	sshArgs[opts.destinationIdx] = sshDestination

	cmd := exec.Command("ssh", sshArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		handleError(err)
	}
}
