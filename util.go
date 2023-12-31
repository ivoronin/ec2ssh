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
	case strings.HasPrefix(dst, "i-"):
		return DstTypeID
	case strings.HasPrefix(dst, "ip-"):
		return DstTypePrivateDNSName
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
	}

	var filterName string

	switch dstType {
	case DstTypeID:
		return awsutil.GetInstanceByID(destination)
	case DstTypePrivateIP:
		filterName = "private-ip-address"
	case DstTypePublicIP:
		filterName = "ip-address"
	case DstTypeIPv6:
		filterName = "ipv6-address"
	case DstTypePrivateDNSName:
		destination += ".*" /* e.g. ip-10-0-0-1.* */
		filterName = "private-dns-name"
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

	return *addr, nil
}
