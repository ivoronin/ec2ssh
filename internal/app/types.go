package app

import (
	"fmt"

	"github.com/ivoronin/ec2ssh/internal/ec2client"
)

// dstTypes maps destination type strings to their enum values.
var dstTypes = map[string]ec2client.DstType{
	"":            ec2client.DstTypeAuto,
	"id":          ec2client.DstTypeID,
	"private_ip":  ec2client.DstTypePrivateIP,
	"public_ip":   ec2client.DstTypePublicIP,
	"ipv6":        ec2client.DstTypeIPv6,
	"private_dns": ec2client.DstTypePrivateDNSName,
	"name_tag":    ec2client.DstTypeNameTag,
}

// addrTypes maps address type strings to their enum values.
var addrTypes = map[string]ec2client.AddrType{
	"":        ec2client.AddrTypeAuto,
	"private": ec2client.AddrTypePrivate,
	"public":  ec2client.AddrTypePublic,
	"ipv6":    ec2client.AddrTypeIPv6,
}

// ParseDstType converts a destination type string to its enum value.
func ParseDstType(s string) (ec2client.DstType, error) {
	dstType, ok := dstTypes[s]
	if !ok {
		return ec2client.DstTypeAuto, fmt.Errorf("%w: %s", ErrUnknownType, s)
	}
	return dstType, nil
}

// ParseAddrType converts an address type string to its enum value.
func ParseAddrType(s string) (ec2client.AddrType, error) {
	addrType, ok := addrTypes[s]
	if !ok {
		return ec2client.AddrTypeAuto, fmt.Errorf("%w: %s", ErrUnknownType, s)
	}
	return addrType, nil
}
