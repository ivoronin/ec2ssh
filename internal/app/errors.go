// Package app provides the core application logic for ec2ssh.
package app

import (
	"errors"
	"fmt"
)

var (
	// ErrUsage is the parent error for all usage/CLI errors.
	ErrUsage              = errors.New("usage error")
	ErrMissingDestination = fmt.Errorf("%w: missing destination", ErrUsage)
	ErrInvalidListColumns = fmt.Errorf("%w: invalid list columns", ErrUsage)
	ErrUnknownType        = fmt.Errorf("%w: unknown type", ErrUsage)
	ErrInvalidOption      = fmt.Errorf("%w: invalid option", ErrUsage)
	ErrExclusiveOptions   = fmt.Errorf("%w: mutually exclusive options", ErrUsage)
)
