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

// InstanceAddr represents a resolved instance address with its type.
type InstanceAddr struct {
	Addr string
	Type AddrType
}

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
func GetInstanceAddr(instance types.Instance, addrType AddrType) (InstanceAddr, error) {
	// Auto mode: try private → public → ipv6
	if addrType == AddrTypeAuto {
		for _, t := range []AddrType{AddrTypePrivate, AddrTypePublic, AddrTypeIPv6} {
			if result, err := GetInstanceAddr(instance, t); err == nil {
				return result, nil
			}
		}
		return InstanceAddr{}, fmt.Errorf("%w: no IP address for instance %s", ErrNoAddress, *instance.InstanceId)
	}

	// Explicit type: lookup address
	addr, name := getAddrByType(instance, addrType)
	if addr == nil {
		return InstanceAddr{}, fmt.Errorf("%w: no %s address for instance %s", ErrNoAddress, name, *instance.InstanceId)
	}
	return InstanceAddr{Addr: *addr, Type: addrType}, nil
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
