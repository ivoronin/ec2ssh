// Package app provides the core application logic for ec2ssh.
package app

import "errors"

// ErrUsage is the sentinel error for all usage/CLI errors.
var ErrUsage = errors.New("usage error")
