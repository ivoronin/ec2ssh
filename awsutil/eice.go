package awsutil

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
)

func SendSSHPublicKey(instance types.Instance, instanceOSUser string, sshPublicKey string) error {
	input := &ec2instanceconnect.SendSSHPublicKeyInput{
		InstanceId:     aws.String(*instance.InstanceId),
		InstanceOSUser: aws.String(instanceOSUser),
		SSHPublicKey:   aws.String(sshPublicKey),
	}

	_, err := awsEC2InstanceConnectClient.SendSSHPublicKey(context.TODO(), input)

	return err
}

func getEICEByID(instanceConnectEndpointID string) (*types.Ec2InstanceConnectEndpoint, error) {
	input := &ec2.DescribeInstanceConnectEndpointsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{"create-complete"},
			},
		},
		InstanceConnectEndpointIds: []string{instanceConnectEndpointID},
	}

	result, err := awsEC2Client.DescribeInstanceConnectEndpoints(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	if len(result.InstanceConnectEndpoints) > 0 {
		return &result.InstanceConnectEndpoints[0], nil
	}

	return nil, fmt.Errorf("unable to find an endpoint with ID=%s: %w", instanceConnectEndpointID, ErrNoMatches)
}

func guessEICEByVPCAndSubnet(vpcID string, subnetID string) (*types.Ec2InstanceConnectEndpoint, error) {
	input := &ec2.DescribeInstanceConnectEndpointsInput{
		Filters: []types.Filter{
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

	var endpoints []types.Ec2InstanceConnectEndpoint

	// Using a paginator to handle potentially paginated results
	paginator := ec2.NewDescribeInstanceConnectEndpointsPaginator(awsEC2Client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}

		/* Look for an endpoint in the same subnet */
		for _, eice := range page.InstanceConnectEndpoints {
			endpoints = append(endpoints, eice)

			if *eice.SubnetId == subnetID {
				return &eice, nil
			}
		}
	}

	/* If we didn't find an endpoint in the same subnet, return the first one */
	if len(endpoints) > 0 {
		return &endpoints[0], nil
	}

	return nil, fmt.Errorf("unable to find an endpoint matching instance vpcID=%s: %w", vpcID, ErrNoMatches)
}

const (
	defaultPresignedURLExpiryTime = 60
	defaultSSHPort                = 22
)

var ErrEICETunnelURI = errors.New("cannot create EICE tunnel URI")

func CreateEICETunnelURI(instance types.Instance, portStr string, eiceID string) (string, error) {
	var err error

	var port int

	if portStr == "" {
		port = defaultSSHPort
	} else {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return "", fmt.Errorf("%w: port is not an integer", ErrEICETunnelURI)
		}

		if port != defaultSSHPort {
			return "", fmt.Errorf("%w: port must be %d", ErrEICETunnelURI, defaultSSHPort)
		}
	}

	if instance.PrivateIpAddress == nil {
		return "", fmt.Errorf("%w: instance %s does not have a private IP address", ErrEICETunnelURI, *instance.InstanceId)
	}

	var eice *types.Ec2InstanceConnectEndpoint
	if eiceID != "" {
		eice, err = getEICEByID(eiceID)
		if err != nil {
			return "", err
		}
	} else {
		eice, err = guessEICEByVPCAndSubnet(*instance.VpcId, *instance.SubnetId)
		if err != nil {
			return "", err
		}
	}

	params := url.Values{}
	params.Add("instanceConnectEndpointId", *eice.InstanceConnectEndpointId)
	params.Add("remotePort", strconv.Itoa(port))
	params.Add("privateIpAddress", *instance.PrivateIpAddress)
	params.Add("X-Amz-Expires", strconv.Itoa(defaultPresignedURLExpiryTime))
	queryString := params.Encode()

	url := fmt.Sprintf("wss://%s/openTunnel?%s", *eice.DnsName, queryString)

	request, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	/* Calculate hash of empty body */
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte{}))
	service := "ec2-instance-connect"
	uri, _, err := awsSigner.PresignHTTP(context.TODO(), awsCredentials, request, hash, service, awsRegion, time.Now())

	return uri, err
}
