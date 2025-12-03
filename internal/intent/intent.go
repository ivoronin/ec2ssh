// Package intent provides intent resolution for ec2ssh based on binary name and flags.
package intent

import "path/filepath"

// Intent represents the operation mode of ec2ssh.
type Intent int

const (
	// IntentSSH connects to an EC2 instance via SSH (default).
	IntentSSH Intent = iota
	// IntentList lists EC2 instances in the region.
	IntentList
	// IntentHelp displays usage information.
	IntentHelp
	// IntentTunnel runs in WebSocket tunnel mode for EICE.
	IntentTunnel
	// IntentSFTP transfers files to/from an EC2 instance via SFTP.
	IntentSFTP
)

// Resolve determines the intent from the binary name and command-line arguments.
// The intent is determined by:
//  1. First argument override (--ssh, --list, --help, --wscat) - wins silently
//  2. Binary name (ec2list -> list, ec2ssh and others -> ssh)
//
// Returns the resolved intent and the remaining arguments (with override flag stripped if present).
func Resolve(binPath string, args []string) (Intent, []string) {
	// Step 1: Check first arg for override (wins silently over binary name)
	if len(args) > 0 {
		switch args[0] {
		case "--list":
			return IntentList, args[1:]
		case "--help", "-h":
			return IntentHelp, args[1:]
		case "--ssh":
			return IntentSSH, args[1:]
		case "--wscat":
			return IntentTunnel, args[1:]
		case "--sftp":
			return IntentSFTP, args[1:]
		}
	}

	// Step 2: Check binary name
	binName := filepath.Base(binPath)
	switch binName {
	case "ec2list":
		return IntentList, args
	case "ec2sftp":
		return IntentSFTP, args
	default:
		// Unknown binary names fall back to SSH (backward compatible)
		return IntentSSH, args
	}
}

// String returns the string representation of the intent.
func (i Intent) String() string {
	switch i {
	case IntentSSH:
		return "ssh"
	case IntentList:
		return "list"
	case IntentHelp:
		return "help"
	case IntentTunnel:
		return "tunnel"
	case IntentSFTP:
		return "sftp"
	default:
		return "unknown"
	}
}
