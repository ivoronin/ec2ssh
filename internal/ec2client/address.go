package ec2client

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// AddrType represents the type of address to use for connection.
// Use a pointer to AddrType where nil means auto-detect.
type AddrType int

const (
	AddrTypePrivate AddrType = iota
	AddrTypePublic
	AddrTypeIPv6
)

// InstanceAddr represents a resolved instance address with its type.
type InstanceAddr struct {
	Addr string
	Type AddrType
}

// UnmarshalText implements encoding.TextUnmarshaler for CLI flag parsing.
// Note: Empty string is not valid - use *AddrType where nil means auto.
func (a *AddrType) UnmarshalText(text []byte) error {
	types := map[string]AddrType{
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
// If addrType is nil, auto-detects by trying public → ipv6 → private.
func GetInstanceAddr(instance types.Instance, addrType *AddrType) (InstanceAddr, error) {
	// nil means auto-detect: try public → ipv6 → private
	if addrType == nil {
		for _, t := range []AddrType{AddrTypePublic, AddrTypeIPv6, AddrTypePrivate} {
			if result, err := GetInstanceAddr(instance, &t); err == nil {
				return result, nil
			}
		}
		return InstanceAddr{}, fmt.Errorf("%w: no IP address for instance %s", ErrNoAddress, *instance.InstanceId)
	}

	// Explicit type: lookup address
	addr, name := getAddrByType(instance, *addrType)
	if addr == nil {
		return InstanceAddr{}, fmt.Errorf("%w: no %s address for instance %s", ErrNoAddress, name, *instance.InstanceId)
	}
	return InstanceAddr{Addr: *addr, Type: *addrType}, nil
}

// GetEICEAddr returns the appropriate address for EICE tunneling.
// EICE can only use private IPv4 or IPv6 (not public).
// If addrType is nil, auto-detects: private IPv4 first, then IPv6.
func GetEICEAddr(instance types.Instance, addrType *AddrType) (InstanceAddr, error) {
	// Explicit type requested
	if addrType != nil {
		if *addrType == AddrTypePublic {
			return InstanceAddr{}, fmt.Errorf("%w: EICE does not support public addresses", ErrNoAddress)
		}
		addr, name := getAddrByType(instance, *addrType)
		if addr == nil {
			return InstanceAddr{}, fmt.Errorf("%w: no %s address for instance %s", ErrNoAddress, name, *instance.InstanceId)
		}
		return InstanceAddr{Addr: *addr, Type: *addrType}, nil
	}

	// Auto-detect: private IPv4 first, then IPv6
	for _, t := range []AddrType{AddrTypePrivate, AddrTypeIPv6} {
		if addr, _ := getAddrByType(instance, t); addr != nil {
			return InstanceAddr{Addr: *addr, Type: t}, nil
		}
	}

	return InstanceAddr{}, fmt.Errorf("%w: no private IPv4 or IPv6 address for instance %s", ErrNoAddress, *instance.InstanceId)
}

func getAddrByType(instance types.Instance, addrType AddrType) (*string, string) {
	switch addrType {
	case AddrTypePrivate:
		return instance.PrivateIpAddress, "private"
	case AddrTypePublic:
		return instance.PublicIpAddress, "public"
	case AddrTypeIPv6:
		return instance.Ipv6Address, "IPv6"
	default:
		return nil, "unknown"
	}
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
