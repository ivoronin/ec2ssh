package ssh

import (
	"fmt"
	"strings"
)

// SFTPTarget is the interface for SFTP destinations.
type SFTPTarget interface {
	sftpTarget()
	Login() string
	Host() string
	Port() string
	Path() string
	SetHost(string)
	SetHostIPv6(string)
	String() string
}

// sftpURLTarget represents sftp://[user@]host[:port][/path]
type sftpURLTarget struct {
	user      string
	hostname  string
	port      string
	path      string
	bracketed bool
}

func (t *sftpURLTarget) sftpTarget()      {}
func (t *sftpURLTarget) Login() string    { return t.user }
func (t *sftpURLTarget) Host() string     { return t.hostname }
func (t *sftpURLTarget) Port() string     { return t.port }
func (t *sftpURLTarget) Path() string     { return t.path }
func (t *sftpURLTarget) SetHost(h string) {
	if h == "" {
		panic("SetHost: empty host")
	}
	t.hostname = h
	t.bracketed = false
}
func (t *sftpURLTarget) SetHostIPv6(h string) {
	if h == "" {
		panic("SetHostIPv6: empty host")
	}
	t.hostname = h
	t.bracketed = true
}
func (t *sftpURLTarget) String() string {
	s := "sftp://"
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
	if t.path != "" {
		s += "/" + t.path
	}
	return s
}

// sftpSimpleTarget represents [user@]host[:path]
type sftpSimpleTarget struct {
	user      string
	hostname  string
	path      string
	bracketed bool
}

func (t *sftpSimpleTarget) sftpTarget()      {}
func (t *sftpSimpleTarget) Login() string    { return t.user }
func (t *sftpSimpleTarget) Host() string     { return t.hostname }
func (t *sftpSimpleTarget) Port() string     { return "" }
func (t *sftpSimpleTarget) Path() string     { return t.path }
func (t *sftpSimpleTarget) SetHost(h string) {
	if h == "" {
		panic("SetHost: empty host")
	}
	t.hostname = h
	t.bracketed = false
}
func (t *sftpSimpleTarget) SetHostIPv6(h string) {
	if h == "" {
		panic("SetHostIPv6: empty host")
	}
	t.hostname = h
	t.bracketed = true
}
func (t *sftpSimpleTarget) String() string {
	s := ""
	if t.user != "" {
		s += t.user + "@"
	}
	if t.bracketed {
		s += "[" + t.hostname + "]"
	} else {
		s += t.hostname
	}
	if t.path != "" {
		s += ":" + t.path
	}
	return s
}

// NewSFTPTarget parses an SFTP destination string into an SFTPTarget.
//
// Supported formats:
//   - Simple: [user@]hostname[:path]
//   - URL: sftp://[user@]hostname[:port][/path]
//
// Returns ErrTarget if the hostname is empty.
func NewSFTPTarget(s string) (SFTPTarget, error) {
	// URL format: sftp://[user@]host[:port][/path]
	if strings.HasPrefix(s, "sftp://") {
		s = strings.TrimPrefix(s, "sftp://")
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
		return &sftpURLTarget{user: user, hostname: host, port: port, path: path, bracketed: bracketed}, nil
	}
	// Simple format: [user@]host[:path] - path is optional
	user, rest, _ := splitUserRest(s)
	host, path, _ := splitHostRest(rest)
	if host == "" {
		return nil, fmt.Errorf("%w: missing hostname", ErrTarget)
	}
	// Detect bracketed IPv6
	bracketed := false
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
		bracketed = true
	}
	return &sftpSimpleTarget{user: user, hostname: host, path: path, bracketed: bracketed}, nil
}
