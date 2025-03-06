package main

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/ivoronin/ec2ssh/awsutil"
)

var (
	allowedListColumns = [...]string{
		"ID", "NAME", "STATE", "TYPE", "AZ", "PRIVATE-IP",
		"PUBLIC-IP", "IPV6", "PRIVATE-DNS", "PUBLIC-DNS",
	}
	defaultListColumns = "ID,NAME,STATE,PRIVATE-IP,PUBLIC-IP"
)

func parseRequestedColumns(requestedColumns string) ([]string, error) {
	if requestedColumns == "" {
		requestedColumns = defaultListColumns
	}

	requestedColumns = strings.ToUpper(requestedColumns)
	requestedColumns = strings.ReplaceAll(requestedColumns, " ", "")

	columns := strings.Split(requestedColumns, ",")

	for _, column := range columns {
		if !slices.Contains(allowedListColumns[:], column) {
			return nil, fmt.Errorf("%w: invalid column %s", ErrArgParse, column)
		}
	}

	return columns, nil
}

const listPadding = 2

func List(options Options) error {
	columns, err := parseRequestedColumns(options.ListColumns)
	if err != nil {
		return err
	}

	instances, err := awsutil.ListInstances()
	if err != nil {
		return fmt.Errorf("unable to list instances: %w", err)
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 1, listPadding, ' ', 0)
	fmt.Fprintln(writer, strings.Join(columns, "\t"))

	for _, instance := range instances {
		state := string(instance.State.Name)
		typ := string(instance.InstanceType)
		values := map[string]*string{
			"ID":          instance.InstanceId,
			"NAME":        GetInstanceName(instance),
			"STATE":       &state,
			"TYPE":        &typ,
			"AZ":          instance.Placement.AvailabilityZone,
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

		fmt.Fprintln(writer, strings.Join(rows, "\t"))
	}

	return writer.Flush()
}
