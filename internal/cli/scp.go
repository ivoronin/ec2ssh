package cli

import "strings"

// SCPOperand represents a parsed SCP operand (source or target).
type SCPOperand struct {
	Login    string // Username (if remote)
	Host     string // EC2 identifier (if remote)
	Path     string // File/directory path
	IsRemote bool   // true if this is a remote operand
}

// findColonSeparator finds the colon that separates host from path in SCP operand.
// Returns the index of the colon, or -1 if this is a local path.
//
// Logic follows OpenSSH's colon() function from misc.c:
//   - Leading colon means it's a filename starting with ':'
//   - '/' before any ':' means it's a local path
//   - '[' starts IPv6 address mode, look for ']:' pattern
//   - Otherwise, first ':' is the separator
func findColonSeparator(s string) int {
	if len(s) == 0 {
		return -1
	}

	// Leading colon is part of filename
	if s[0] == ':' {
		return -1
	}

	inBrackets := s[0] == '['

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '@':
			// user@[ipv6] - start bracket mode
			if i+1 < len(s) && s[i+1] == '[' {
				inBrackets = true
			}
		case ']':
			// End of IPv6, check for following colon
			if inBrackets && i+1 < len(s) && s[i+1] == ':' {
				return i + 1
			}
		case ':':
			// Found colon outside brackets
			if !inBrackets {
				return i
			}
		case '/':
			// Slash before colon means local path
			return -1
		}
	}

	return -1
}

// ParseSCPOperand parses a single SCP operand (source or target).
// Uses OpenSSH-compatible logic to distinguish local from remote paths.
//
// Supports formats:
//   - Local: /path, ./path, path, ~/path, path/with:colon
//   - Remote: host:path, user@host:path, [ipv6]:path, user@[ipv6]:path
func ParseSCPOperand(operand string) SCPOperand {
	colonIdx := findColonSeparator(operand)

	if colonIdx == -1 {
		// Local path
		return SCPOperand{Path: operand, IsRemote: false}
	}

	// Remote: split at colon
	loginHost := operand[:colonIdx]
	path := operand[colonIdx+1:]

	// Extract login and host, handling IPv6 brackets
	login, host := parseUserHost(loginHost)

	return SCPOperand{
		Login:    login,
		Host:     host,
		Path:     path,
		IsRemote: true,
	}
}

// parseUserHost extracts user and host from "[user@]host" string.
// Strips IPv6 brackets if present.
func parseUserHost(s string) (login, host string) {
	login, host = parseLoginHost(s)

	// Strip IPv6 brackets from host
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}

	return login, host
}

// SCPParsedOperands holds the result of parsing two SCP operands.
type SCPParsedOperands struct {
	Login      string // Username from remote operand
	Host       string // EC2 instance identifier from remote operand
	RemotePath string // Path on remote
	LocalPath  string // Path on local machine
	IsUpload   bool   // true = local→remote, false = remote→local
}

// ParseSCPOperands parses and validates two SCP operands.
// Returns error if:
//   - Not exactly 2 operands
//   - Both operands are local (no remote)
//   - Both operands are remote (multiple remotes not supported)
//   - Remote path is empty after ':'
//   - Remote host is empty
func ParseSCPOperands(operands []string) (SCPParsedOperands, error) {
	if len(operands) < 2 {
		return SCPParsedOperands{}, ErrSCPTooFewOperands
	}

	if len(operands) > 2 {
		return SCPParsedOperands{}, ErrSCPTooManyOperands
	}

	source := ParseSCPOperand(operands[0])
	target := ParseSCPOperand(operands[1])

	// Validate: exactly one must be remote
	if !source.IsRemote && !target.IsRemote {
		return SCPParsedOperands{}, ErrSCPNoRemote
	}

	if source.IsRemote && target.IsRemote {
		return SCPParsedOperands{}, ErrSCPMultipleRemotes
	}

	var result SCPParsedOperands

	if source.IsRemote {
		// Download: remote → local
		result.IsUpload = false
		result.Login = source.Login
		result.Host = source.Host
		result.RemotePath = source.Path
		result.LocalPath = target.Path
	} else {
		// Upload: local → remote
		result.IsUpload = true
		result.Login = target.Login
		result.Host = target.Host
		result.RemotePath = target.Path
		result.LocalPath = source.Path
	}

	// Validate remote components
	if result.Host == "" {
		return SCPParsedOperands{}, ErrSCPEmptyHost
	}

	if result.RemotePath == "" {
		return SCPParsedOperands{}, ErrSCPEmptyPath
	}

	return result, nil
}
