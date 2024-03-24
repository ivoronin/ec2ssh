package main

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/ivoronin/ec2ssh/awsutil"
)

var ErrNoAddress = errors.New("no address found")

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

func GetInstance(dstType DstType, destination string) (types.Instance, error) {
	if dstType == DstTypeAuto {
		dstType = GuessDestinationType(destination)

		DebugLogger.Printf("guessed destination type %d for %s", dstType, destination)
	}

	var filterName string

	switch dstType {
	case DstTypeID:
		instance, err := awsutil.GetInstanceByID(destination)

		return instance, err
	case DstTypePrivateIP:
		filterName = "private-ip-address"
	case DstTypePublicIP:
		filterName = "ip-address"
	case DstTypeIPv6:
		filterName = "ipv6-address"
	case DstTypePrivateDNSName:
		filterName = "private-dns-name"

		if !strings.Contains(destination, ".") {
			destination += ".*" /* e.g. ip-10-0-0-1.* */
		}
	case DstTypeNameTag:
		filterName = "tag:Name"
	case DstTypeAuto:
	default:
		// Should never happen
		panic(dstType)
	}

	return awsutil.GetInstanceByFilter(filterName, destination)
}

func GetInstanceAddr(instance types.Instance, addrType AddrType) (string, error) {
	var addr *string

	var typeStr string

	switch addrType {
	case AddrTypeAuto:
		switch {
		case instance.PrivateIpAddress != nil:
			addr = instance.PrivateIpAddress
			typeStr = "private"
		case instance.PublicIpAddress != nil:
			addr = instance.PublicIpAddress
			typeStr = "public"
		case instance.Ipv6Address != nil:
			addr = instance.Ipv6Address
			typeStr = "IPv6"
		}
	case AddrTypePrivate:
		addr = instance.PrivateIpAddress
		typeStr = "private"
	case AddrTypePublic:
		addr = instance.PublicIpAddress
		typeStr = "public"
	case AddrTypeIPv6:
		addr = instance.Ipv6Address
		typeStr = "IPv6"
	}

	if addr == nil {
		return "", fmt.Errorf("%w: no %s IP address found for instance ID %s", ErrNoAddress, typeStr, *instance.InstanceId)
	}

	DebugLogger.Printf("using %s IP address %s for instance ID %s", typeStr, *addr, *instance.InstanceId)

	return *addr, nil
}
