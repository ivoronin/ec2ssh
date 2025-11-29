package ec2

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// DstType represents the type of destination identifier.
type DstType int

const (
	DstTypeAuto DstType = iota
	DstTypeID
	DstTypePrivateIP
	DstTypePublicIP
	DstTypeIPv6
	DstTypePrivateDNSName
	DstTypeNameTag
)

// ErrNoAddress is returned when an instance doesn't have the requested address type.
var ErrNoAddress = errors.New("no address found")

// ErrNoMatches is returned when no instances match the search criteria.
var ErrNoMatches = errors.New("no matching instances found")

// GetInstanceByID retrieves an instance by its ID.
func (c *Client) GetInstanceByID(instanceID string) (types.Instance, error) {
	c.logger.Printf("searching for instance by ID %s", instanceID)

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}

	instance, err := c.getFirstMatchingInstance(input)
	if err != nil {
		return types.Instance{}, fmt.Errorf("unable to find an instance with ID=%s: %w", instanceID, err)
	}

	return instance, nil
}

// GetRunningInstanceByFilter retrieves a running instance matching the given filter.
func (c *Client) GetRunningInstanceByFilter(filterName, filterValue string) (types.Instance, error) {
	c.logger.Printf("searching for instance by %s=%s", filterName, filterValue)

	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String(filterName),
				Values: []string{filterValue},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running"},
			},
		},
	}

	instance, err := c.getFirstMatchingInstance(input)
	if err != nil {
		return types.Instance{}, fmt.Errorf("unable to find a runnning instance with %s=%s: %w", filterName, filterValue, err)
	}

	return instance, nil
}

// ListInstances returns all instances in the region.
func (c *Client) ListInstances() ([]types.Instance, error) {
	c.logger.Printf("listing all instances")

	input := &ec2.DescribeInstancesInput{}

	result, err := c.ec2Client.DescribeInstances(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	var instances []types.Instance

	for _, reservation := range result.Reservations {
		instances = append(instances, reservation.Instances...)
	}

	return instances, nil
}

func (c *Client) getFirstMatchingInstance(input *ec2.DescribeInstancesInput) (types.Instance, error) {
	result, err := c.ec2Client.DescribeInstances(context.TODO(), input)
	if err != nil {
		return types.Instance{}, err
	}

	c.logger.Printf("found %d reservations", len(result.Reservations))

	for rsvIdx, reservation := range result.Reservations {
		c.logger.Printf("found %d instances in reservation %d", len(reservation.Instances), rsvIdx)

		for _, instance := range reservation.Instances {
			c.logger.Printf("selected first matching instance %s", *instance.InstanceId)

			return instance, nil
		}
	}

	return types.Instance{}, fmt.Errorf("%w in %s", ErrNoMatches, c.region)
}

// GuessDestinationType infers the destination type from the destination string.
func GuessDestinationType(dst string) DstType {
	switch {
	case strings.HasPrefix(dst, "ip-"),
		strings.HasSuffix(dst, ".ec2.internal"),
		strings.HasSuffix(dst, ".compute.internal"):
		return DstTypePrivateDNSName
	case strings.HasPrefix(dst, "i-"):
		return DstTypeID
	case net.ParseIP(dst) != nil:
		addr := net.ParseIP(dst)
		if addr.To4() != nil {
			if addr.IsPrivate() {
				return DstTypePrivateIP
			}

			return DstTypePublicIP
		}

		return DstTypeIPv6
	default:
		return DstTypeNameTag
	}
}

// GetInstance retrieves an instance using the specified destination type and value.
func (c *Client) GetInstance(dstType DstType, destination string) (types.Instance, error) {
	if dstType == DstTypeAuto {
		dstType = GuessDestinationType(destination)
		c.logger.Printf("guessed destination type %d for %s", dstType, destination)
	}

	var filterName string

	switch dstType {
	case DstTypeID:
		return c.GetInstanceByID(destination)
	case DstTypePrivateIP:
		filterName = "private-ip-address"
	case DstTypePublicIP:
		filterName = "ip-address"
	case DstTypeIPv6:
		filterName = "ipv6-address"
	case DstTypePrivateDNSName:
		filterName = "private-dns-name"

		if !strings.Contains(destination, ".") {
			destination += ".*" // e.g. ip-10-0-0-1.*
		}
	case DstTypeNameTag:
		filterName = "tag:Name"
	case DstTypeAuto:
		fallthrough
	default:
		// Should never happen
		panic(dstType)
	}

	return c.GetRunningInstanceByFilter(filterName, destination)
}
