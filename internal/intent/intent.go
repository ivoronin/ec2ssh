// Package intent provides intent resolution for ec2ssh based on binary name and flags.
package intent

import "path/filepath"

// Intent represents the operation mode of ec2ssh.
type Intent int

const (
	// IntentHelp displays usage information (hidden internal).
	IntentHelp Intent = iota
	// IntentVersion displays the version and exits (hidden internal).
	IntentVersion
	// IntentSSH connects to an EC2 instance via SSH (default).
	IntentSSH
	// IntentSCP copies files to/from an EC2 instance via SCP.
	IntentSCP
	// IntentSFTP transfers files to/from an EC2 instance via SFTP.
	IntentSFTP
	// IntentEICETunnel runs in WebSocket tunnel mode for EICE (hidden internal).
	IntentEICETunnel
	// IntentSSMSession starts an SSM Session Manager shell.
	IntentSSMSession
	// IntentSSMTunnel runs in SSM tunnel mode for SSH over SSM (hidden internal).
	IntentSSMTunnel
	// IntentList lists EC2 instances in the region.
	IntentList
)

// Resolve determines the intent from the binary name and command-line arguments.
// The intent is determined by:
//  1. First argument override (--ssh, --list, --help, --eice-tunnel) - wins silently
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
		case "--eice-tunnel":
			return IntentEICETunnel, args[1:]
		case "--sftp":
			return IntentSFTP, args[1:]
		case "--scp":
			return IntentSCP, args[1:]
		case "--version":
			return IntentVersion, args[1:]
		case "--ssm":
			return IntentSSMSession, args[1:]
		case "--ssm-tunnel":
			return IntentSSMTunnel, args[1:]
		}
	}

	// Step 2: Check binary name
	binName := filepath.Base(binPath)
	switch binName {
	case "ec2list":
		return IntentList, args
	case "ec2sftp":
		return IntentSFTP, args
	case "ec2scp":
		return IntentSCP, args
	case "ec2ssm":
		return IntentSSMSession, args
	default:
		// Unknown binary names fall back to SSH (backward compatible)
		return IntentSSH, args
	}
}

// String returns the string representation of the intent.
func (i Intent) String() string {
	switch i {
	case IntentHelp:
		return "help"
	case IntentVersion:
		return "version"
	case IntentSSH:
		return "ssh"
	case IntentSCP:
		return "scp"
	case IntentSFTP:
		return "sftp"
	case IntentEICETunnel:
		return "eice-tunnel"
	case IntentSSMSession:
		return "ssm"
	case IntentSSMTunnel:
		return "ssm-tunnel"
	case IntentList:
		return "list"
	default:
		return "unknown"
	}
}
