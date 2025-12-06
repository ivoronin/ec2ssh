package ec2client

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

const defaultPresignedURLExpiryTime = 60

func (c *Client) getEICEByID(instanceConnectEndpointID string) (*types.Ec2InstanceConnectEndpoint, error) {
	c.logger.Printf("searching for endpoint by ID %s", instanceConnectEndpointID)

	input := &ec2.DescribeInstanceConnectEndpointsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{"create-complete"},
			},
		},
		InstanceConnectEndpointIds: []string{instanceConnectEndpointID},
	}

	result, err := c.ec2Client.DescribeInstanceConnectEndpoints(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	c.logger.Printf("found %d endpoints", len(result.InstanceConnectEndpoints))

	if len(result.InstanceConnectEndpoints) > 0 {
		eice := result.InstanceConnectEndpoints[0]

		c.logger.Printf("selected first matching endpoint %s", *eice.InstanceConnectEndpointId)

		return &eice, nil
	}

	return nil, fmt.Errorf("unable to find an endpoint with ID=%s: %w", instanceConnectEndpointID, ErrNoMatches)
}

// GuessEICEByVPCAndSubnet finds an EICE endpoint in the given VPC, preferring one in the same subnet.
func (c *Client) GuessEICEByVPCAndSubnet(vpcID string, subnetID string) (*types.Ec2InstanceConnectEndpoint, error) {
	c.logger.Printf("searching for EICE by vpcID %s and subnetID %s", vpcID, subnetID)

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
	paginator := ec2.NewDescribeInstanceConnectEndpointsPaginator(c.ec2Client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}

		c.logger.Printf("found %d endpoints", len(page.InstanceConnectEndpoints))

		// Look for an endpoint in the same subnet
		for _, eice := range page.InstanceConnectEndpoints {
			endpoints = append(endpoints, eice)

			if *eice.SubnetId == subnetID {
				eiceID := *eice.InstanceConnectEndpointId
				c.logger.Printf("selected first endpoint matching instance vpc and subnet: %s", eiceID)

				return &eice, nil
			}
		}
	}

	// If we didn't find an endpoint in the same subnet, return the first one
	if len(endpoints) > 0 {
		c.logger.Printf("found endpoint ID %s matching instance vpc", *endpoints[0].InstanceConnectEndpointId)

		return &endpoints[0], nil
	}

	return nil, fmt.Errorf("unable to find an endpoint matching instance vpcID=%s: %w", vpcID, ErrNoMatches)
}

// CreateEICETunnelURI creates a signed WebSocket tunnel URI for EICE connection.
func (c *Client) CreateEICETunnelURI(privateIP, portStr, eiceID string) (string, error) {
	c.logger.Printf("creating EICE tunnel URI for %s via %s", privateIP, eiceID)

	eice, err := c.getEICEByID(eiceID)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Add("instanceConnectEndpointId", *eice.InstanceConnectEndpointId)
	params.Add("remotePort", portStr)
	params.Add("privateIpAddress", privateIP)
	params.Add("X-Amz-Expires", strconv.Itoa(defaultPresignedURLExpiryTime))
	queryString := params.Encode()

	unsignedURL := fmt.Sprintf("wss://%s/openTunnel?%s", *eice.DnsName, queryString)

	request, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, unsignedURL, nil)
	if err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte{}))
	service := "ec2-instance-connect"
	uri, _, err := c.signer.PresignHTTP(context.TODO(), c.credentials, request, hash, service, c.region, time.Now())

	c.logger.Printf("created EICE tunnel URI %s", uri)

	return uri, err
}
