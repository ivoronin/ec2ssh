package ssh

import (
	"fmt"
	"strings"
)

// SCPTarget is the interface for SCP destinations.
type SCPTarget interface {
	scpTarget()
	Login() string
	Host() string
	Port() string
	Path() string
	SetHost(string)
	SetHostIPv6(string)
	String() string
}

// scpURLTarget represents scp://[user@]host[:port]/path
type scpURLTarget struct {
	user      string
	hostname  string
	port      string
	path      string
	bracketed bool
}

func (t *scpURLTarget) scpTarget()    {}
func (t *scpURLTarget) Login() string { return t.user }
func (t *scpURLTarget) Host() string  { return t.hostname }
func (t *scpURLTarget) Port() string  { return t.port }
func (t *scpURLTarget) Path() string  { return "/" + t.path }
func (t *scpURLTarget) SetHost(h string) {
	if h == "" {
		panic("SetHost: empty host")
	}
	t.hostname = h
	t.bracketed = false
}
func (t *scpURLTarget) SetHostIPv6(h string) {
	if h == "" {
		panic("SetHostIPv6: empty host")
	}
	t.hostname = h
	t.bracketed = true
}
func (t *scpURLTarget) String() string {
	s := "scp://"
	if t.user != "" {
		s += t.user + "@"
	}
	if t.bracketed {
		s += "[" + t.hostname + "]"
	} else {
		s += t.hostname
	}
	if t.port != "" {
		s += ":" + t.port
	}
	s += "/" + t.path
	return s
}

// scpSimpleTarget represents [user@]host:path (colon required)
type scpSimpleTarget struct {
	user      string
	hostname  string
	path      string
	bracketed bool
}

func (t *scpSimpleTarget) scpTarget()    {}
func (t *scpSimpleTarget) Login() string { return t.user }
func (t *scpSimpleTarget) Host() string  { return t.hostname }
func (t *scpSimpleTarget) Port() string  { return "" }
func (t *scpSimpleTarget) Path() string  { return t.path }
func (t *scpSimpleTarget) SetHost(h string) {
	if h == "" {
		panic("SetHost: empty host")
	}
	t.hostname = h
	t.bracketed = false
}
func (t *scpSimpleTarget) SetHostIPv6(h string) {
	if h == "" {
		panic("SetHostIPv6: empty host")
	}
	t.hostname = h
	t.bracketed = true
}
func (t *scpSimpleTarget) String() string {
	s := ""
	if t.user != "" {
		s += t.user + "@"
	}
	if t.bracketed {
		s += "[" + t.hostname + "]"
	} else {
		s += t.hostname
	}
	s += ":" + t.path
	return s
}

// IsLocalPath returns true if the string represents a local path (not a remote target).
// Matches OpenSSH's colon() function behavior from misc.c.
//
// A path is considered local if:
//   - It's empty (returns true)
//   - It starts with ':' (colon is part of filename)
//   - It contains '/' before any unbracketed ':' (path separator wins)
//   - It has no unbracketed ':' at all
//
// IPv6 addresses in brackets [addr] are handled: only "]: triggers remote detection.
func IsLocalPath(s string) bool {
	if len(s) == 0 {
		return true
	}

	// Leading colon is part of filename (OpenSSH behavior)
	if s[0] == ':' {
		return true
	}

	// Bracket mode: only if string starts with '['
	inBracket := s[0] == '['

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '@':
			// Enable bracket mode if next char is '[' (user@[ipv6])
			if i+1 < len(s) && s[i+1] == '[' {
				inBracket = true
			}
		case ']':
			// Check for ]: while in bracket mode → remote
			if inBracket && i+1 < len(s) && s[i+1] == ':' {
				return false
			}
		case ':':
			// Colon outside brackets → remote
			if !inBracket {
				return false
			}
		case '/':
			// Slash before colon → local path
			return true
		}
	}

	return true // No colon found
}

// NewSCPTarget parses an SCP destination string into an SCPTarget.
//
// Supported formats:
//   - Simple: [user@]hostname:path (colon required)
//   - URL: scp://[user@]hostname[:port]/path
//
// Returns ErrTarget if the hostname is empty or (for simple format) the colon is missing.
//
// Important: Call IsLocalPath(s) first to check if it's a local path!
func NewSCPTarget(s string) (SCPTarget, error) {
	// URL format: scp://[user@]host[:port]/path
	if strings.HasPrefix(s, "scp://") {
		s = strings.TrimPrefix(s, "scp://")
		hostPart, path := s, ""
		if idx := strings.Index(s, "/"); idx != -1 {
			hostPart, path = s[:idx], s[idx+1:]
		}
		user, rest, _ := splitUserRest(hostPart)
		host, port, _ := splitHostRest(rest)
		if host == "" {
			return nil, fmt.Errorf("%w: missing hostname", ErrTarget)
		}
		// Detect bracketed IPv6
		bracketed := false
		if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
			host = host[1 : len(host)-1]
			bracketed = true
		}
		return &scpURLTarget{user: user, hostname: host, port: port, path: path, bracketed: bracketed}, nil
	}
	// Simple format: [user@]host:path
	user, rest, _ := splitUserRest(s)
	host, path, ok := splitHostRest(rest)
	if !ok {
		return nil, fmt.Errorf("%w: scp target requires colon", ErrTarget)
	}
	if host == "" {
		return nil, fmt.Errorf("%w: missing hostname", ErrTarget)
	}
	// Detect bracketed IPv6
	bracketed := false
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
		bracketed = true
	}
	return &scpSimpleTarget{user: user, hostname: host, path: path, bracketed: bracketed}, nil
}
