package tunnel

import (
	"context"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/mmmorris1975/ssm-session-client/ssmclient"
)

// RunSSM starts an SSM SSH tunnel using the provided tunnel info.
// The tunnelInfo format is: "instanceID:port:region:profile"
func RunSSM(tunnelInfo string) error {
	// Parse tunnel info: instanceID:port:region:profile
	parts := strings.SplitN(tunnelInfo, ":", 4)
	instanceID := parts[0]

	port := 22
	if len(parts) > 1 && parts[1] != "" {
		if p, err := strconv.Atoi(parts[1]); err == nil {
			port = p
		}
	}

	var region, profile string
	if len(parts) > 2 {
		region = parts[2]
	}
	if len(parts) > 3 {
		profile = parts[3]
	}

	// Build AWS config
	optFns := make([]func(*config.LoadOptions) error, 0)
	if region != "" {
		optFns = append(optFns, config.WithRegion(region))
	}
	if profile != "" {
		optFns = append(optFns, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), optFns...)
	if err != nil {
		return err
	}

	return ssmclient.SSHSession(cfg, &ssmclient.PortForwardingInput{
		Target:     instanceID,
		RemotePort: port,
	})
}
