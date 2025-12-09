package ssh

import (
	"fmt"
	"strings"
)

// SSHTarget is the interface for SSH destinations.
type SSHTarget interface {
	sshTarget()
	Login() string
	Host() string
	Port() string
	SetHost(string)
	SetHostIPv6(string)
	String() string
}

// sshURLTarget represents ssh://[user@]host[:port]
type sshURLTarget struct {
	user      string
	hostname  string
	port      string
	bracketed bool
}

func (t *sshURLTarget) sshTarget()    {}
func (t *sshURLTarget) Login() string { return t.user }
func (t *sshURLTarget) Host() string  { return t.hostname }
func (t *sshURLTarget) Port() string  { return t.port }
func (t *sshURLTarget) SetHost(h string) {
	if h == "" {
		panic("SetHost: empty host")
	}
	t.hostname = h
	t.bracketed = false
}
func (t *sshURLTarget) SetHostIPv6(h string) {
	if h == "" {
		panic("SetHostIPv6: empty host")
	}
	t.hostname = h
	t.bracketed = true
}
func (t *sshURLTarget) String() string {
	s := "ssh://"
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
	return s
}

// sshSimpleTarget represents [user@]host (split at last @, mimics OpenSSH)
type sshSimpleTarget struct {
	user     string
	hostname string
}

func (t *sshSimpleTarget) sshTarget()    {}
func (t *sshSimpleTarget) Login() string { return t.user }
func (t *sshSimpleTarget) Host() string  { return t.hostname }
func (t *sshSimpleTarget) Port() string  { return "" }
func (t *sshSimpleTarget) SetHost(h string) {
	if h == "" {
		panic("SetHost: empty host")
	}
	t.hostname = h
}
func (t *sshSimpleTarget) SetHostIPv6(h string) {
	t.SetHost(h) // Simple SSH format doesn't need brackets
}
func (t *sshSimpleTarget) String() string {
	if t.user != "" {
		return t.user + "@" + t.hostname
	}
	return t.hostname
}

// NewSSHTarget parses an SSH destination string into an SSHTarget.
//
// Supported formats:
//   - Simple: [user@]hostname
//   - URL: ssh://[user@]hostname[:port]
//
// Returns ErrTarget if the hostname is empty.
func NewSSHTarget(s string) (SSHTarget, error) {
	// URL format: ssh://[user@]host[:port]
	if strings.HasPrefix(s, "ssh://") {
		s = strings.TrimPrefix(s, "ssh://")
		user, rest, _ := splitUserRest(s)
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
		return &sshURLTarget{user: user, hostname: host, port: port, bracketed: bracketed}, nil
	}
	// Simple format: [user@]host
	user, host, _ := splitUserRest(s)
	if host == "" {
		return nil, fmt.Errorf("%w: missing hostname", ErrTarget)
	}
	return &sshSimpleTarget{user: user, hostname: host}, nil
}
