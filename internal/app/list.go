package app

import (
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/ivoronin/argsieve"
	"github.com/ivoronin/ec2ssh/internal/awsclient"
	"github.com/ivoronin/ec2ssh/internal/ec2client"
)

var (
	allowedListColumns = []string{
		"ID", "NAME", "STATE", "TYPE", "AZ", "PRIVATE-IP",
		"PUBLIC-IP", "IPV6", "PRIVATE-DNS", "PUBLIC-DNS",
	}
	defaultListColumns = "ID,NAME,STATE,PRIVATE-IP,PUBLIC-IP"
)

const listPadding = 2

// ListOptions holds the parsed configuration for listing instances.
type ListOptions struct {
	Region  string `long:"region"`
	Profile string `long:"profile"`
	Columns string `long:"list-columns"`
	Debug   bool   `long:"debug"`
}

// NewListOptions creates ListOptions from command-line arguments.
func NewListOptions(args []string) (*ListOptions, error) {
	var options ListOptions

	positional, err := argsieve.Parse(&options, args)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUsage, err)
	}

	// List doesn't accept positional arguments
	if len(positional) > 0 {
		return nil, fmt.Errorf("%w: unexpected argument %s", ErrUsage, positional[0])
	}

	return &options, nil
}

// RunList executes the list intent with the given arguments.
func RunList(args []string) error {
	options, err := NewListOptions(args)
	if err != nil {
		return err
	}

	columns, err := parseListColumns(options.Columns)
	if err != nil {
		return fmt.Errorf("%w: invalid list columns: %v", ErrUsage, err)
	}

	logger := log.New(io.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	if options.Debug {
		logger.SetOutput(os.Stderr)
	}

	cfg, err := awsclient.LoadConfig(options.Region, options.Profile, logger)
	if err != nil {
		return err
	}

	client, err := ec2client.NewClient(cfg, logger)
	if err != nil {
		return err
	}

	instances, err := client.ListInstances()
	if err != nil {
		return fmt.Errorf("unable to list instances: %w", err)
	}

	return writeInstanceList(os.Stdout, instances, columns)
}

func parseListColumns(requestedColumns string) ([]string, error) {
	if requestedColumns == "" {
		requestedColumns = defaultListColumns
	}

	requestedColumns = strings.ToUpper(requestedColumns)
	requestedColumns = strings.ReplaceAll(requestedColumns, " ", "")

	columns := strings.Split(requestedColumns, ",")

	for _, column := range columns {
		if !slices.Contains(allowedListColumns, column) {
			return nil, fmt.Errorf("invalid column %s", column)
		}
	}

	return columns, nil
}

func writeInstanceList(w io.Writer, instances []types.Instance, columns []string) error {
	writer := tabwriter.NewWriter(w, 0, 1, listPadding, ' ', 0)
	_, _ = fmt.Fprintln(writer, strings.Join(columns, "\t"))

	for _, instance := range instances {
		var state string
		if instance.State != nil {
			state = string(instance.State.Name)
		}

		typ := string(instance.InstanceType)

		var az *string
		if instance.Placement != nil {
			az = instance.Placement.AvailabilityZone
		}

		values := map[string]*string{
			"ID":          instance.InstanceId,
			"NAME":        ec2client.GetInstanceName(instance),
			"STATE":       &state,
			"TYPE":        &typ,
			"AZ":          az,
			"PRIVATE-IP":  instance.PrivateIpAddress,
			"PUBLIC-IP":   instance.PublicIpAddress,
			"IPV6":        instance.Ipv6Address,
			"PRIVATE-DNS": instance.PrivateDnsName,
			"PUBLIC-DNS":  instance.PublicDnsName,
		}

		var rows []string

		for _, column := range columns {
			value := "-"
			if values[column] != nil && *(values[column]) != "" {
				value = *values[column]
			}

			rows = append(rows, value)
		}

		_, _ = fmt.Fprintln(writer, strings.Join(rows, "\t"))
	}

	return writer.Flush()
}
