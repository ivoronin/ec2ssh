// Package ssh provides command-line argument parsing for SSH/SCP/SFTP targets.
package ssh

import (
	"errors"
	"strings"
)

// ErrTarget indicates a target parsing error.
var ErrTarget = errors.New("invalid target")

// Target is the base interface for all SSH/SCP/SFTP targets.
// Used by baseSSHSession for AWS operations and building command args.
type Target interface {
	Login() string
	Host() string
	SetHost(string)
	SetHostIPv6(string)
	String() string
}

// splitUserRest splits "user@rest" at last @ (mimics OpenSSH strrchr).
// Returns (user, rest, true) if @ found, or ("", s, false) if not.
func splitUserRest(s string) (user, rest string, ok bool) {
	if idx := strings.LastIndex(s, "@"); idx != -1 {
		return s[:idx], s[idx+1:], true
	}
	return "", s, false
}

// splitHostRest splits "host:rest" handling IPv6 brackets.
// Returns (host, rest, true) if colon found, or (s, "", false) if not.
func splitHostRest(s string) (host, rest string, ok bool) {
	if strings.HasPrefix(s, "[") {
		// IPv6: [addr]:rest - find "]:"
		if idx := strings.Index(s, "]:"); idx != -1 {
			return s[:idx+1], s[idx+2:], true
		}
		return s, "", false // [addr] with no path
	}
	// Regular: host:rest - find first ":"
	if idx := strings.Index(s, ":"); idx != -1 {
		return s[:idx], s[idx+1:], true
	}
	return s, "", false
}
