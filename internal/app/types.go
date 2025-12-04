package app

import (
	"fmt"

	"github.com/ivoronin/ec2ssh/internal/ec2"
)

// dstTypes maps destination type strings to their enum values.
var dstTypes = map[string]ec2.DstType{
	"":            ec2.DstTypeAuto,
	"id":          ec2.DstTypeID,
	"private_ip":  ec2.DstTypePrivateIP,
	"public_ip":   ec2.DstTypePublicIP,
	"ipv6":        ec2.DstTypeIPv6,
	"private_dns": ec2.DstTypePrivateDNSName,
	"name_tag":    ec2.DstTypeNameTag,
}

// addrTypes maps address type strings to their enum values.
var addrTypes = map[string]ec2.AddrType{
	"":        ec2.AddrTypeAuto,
	"private": ec2.AddrTypePrivate,
	"public":  ec2.AddrTypePublic,
	"ipv6":    ec2.AddrTypeIPv6,
}

// ParseDstType converts a destination type string to its enum value.
func ParseDstType(s string) (ec2.DstType, error) {
	dstType, ok := dstTypes[s]
	if !ok {
		return ec2.DstTypeAuto, fmt.Errorf("%w: %s", ErrUnknownType, s)
	}
	return dstType, nil
}

// ParseAddrType converts an address type string to its enum value.
func ParseAddrType(s string) (ec2.AddrType, error) {
	addrType, ok := addrTypes[s]
	if !ok {
		return ec2.AddrTypeAuto, fmt.Errorf("%w: %s", ErrUnknownType, s)
	}
	return addrType, nil
}
