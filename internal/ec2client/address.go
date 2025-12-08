package ec2client

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// AddrType represents the type of address to use for connection.
type AddrType int

const (
	AddrTypeAuto AddrType = iota
	AddrTypePrivate
	AddrTypePublic
	AddrTypeIPv6
)

// UnmarshalText implements encoding.TextUnmarshaler for CLI flag parsing.
func (a *AddrType) UnmarshalText(text []byte) error {
	types := map[string]AddrType{
		"":        AddrTypeAuto,
		"private": AddrTypePrivate,
		"public":  AddrTypePublic,
		"ipv6":    AddrTypeIPv6,
	}
	t, ok := types[string(text)]
	if !ok {
		return fmt.Errorf("unknown address type: %s", text)
	}
	*a = t
	return nil
}

// GetInstanceAddr returns the appropriate IP address for an instance.
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

// GetInstanceName returns the Name tag value for an instance.
func GetInstanceName(instance types.Instance) *string {
	return getInstanceTagValue(instance, "Name")
}

func getInstanceTagValue(instance types.Instance, tagKey string) *string {
	for _, tag := range instance.Tags {
		if *tag.Key == tagKey {
			return tag.Value
		}
	}

	return nil
}
